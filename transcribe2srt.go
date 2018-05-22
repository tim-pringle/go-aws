package transcribe2srt

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/transcribeservice"
)

var (
	//ErrJobnameNotProvided represents when the content of the request is empty
	ErrJobnameNotProvided = errors.New("No job number has been provided HTTP body")
	//ErrTranscribeRunning indicates that a Transcription job is still not completed
	ErrTranscribeRunning = errors.New("The transcription job is running")
	//ErrTranscribeFailure signifies an unsuccessful Tanscription job
	ErrTranscribeFailure = errors.New("The transcription job has failed")
	//ErrJobDoesntExist advises there is no Transcription job number as provided in the request
	ErrJobDoesntExist = errors.New("The transcripting job does not exist")
)

//Convert receives a Transcribe job identifier, retrieves the job data and then converts
//the transcription content into an srt format, which is returned to the caller
func Convert(jobname string) (string, error) {

	// stdout and stderr are sent to AWS CloudWatch Logs
	log.Printf("Processing conversion request")

	sess, _ := session.NewSessionWithOptions(session.Options{
		Config:  aws.Config{Region: aws.String("eu-west-1")},
		Profile: "development",
	})

	log.Printf("Job name : %s", jobname)
	log.Printf("Creating new session")
	transcriber := transcribeservice.New(sess)

	log.Printf("Getting transcription job")
	transcriptionjobinput := transcribeservice.GetTranscriptionJobInput{
		TranscriptionJobName: &jobname,
	}
	transcriptionjoboutput, err := transcriber.GetTranscriptionJob(&transcriptionjobinput)

	if err != nil {
		ErrMsg := errors.New(err.Error())
		log.Printf("%s", err.Error())
		return "", ErrMsg
	}

	strStatus := *(transcriptionjoboutput.TranscriptionJob.TranscriptionJobStatus)
	log.Printf("Job status is %s", strStatus)

	if strStatus == "FAILED" {
		return "", ErrTranscribeFailure
	}
	if strStatus == "IN_PROGRESS" {
		return "", ErrTranscribeRunning
	}

	var uri *string
	uri = transcriptionjoboutput.TranscriptionJob.Transcript.TranscriptFileUri
	log.Printf("URI is %s", *uri)

	response, _ := http.Get(*uri)
	defer response.Body.Close()
	body, _ := ioutil.ReadAll(response.Body)

	//If there's an error, print the error
	if err != nil {
		fmt.Println(err)
	}

	// initialize our variable to hold the json
	var awstranscript Awstranscript

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'awstranscript' which we defined above
	json.Unmarshal(body, &awstranscript)

	var transcription []Item
	transcription = awstranscript.Results.Items

	var index, sequence int = 0, 0
	var srtinfo, subdetail, subtitle, sttime, classification, text, entime string
	var strlen int
	var firstrow bool

	for index = 0; index < len(transcription); {
		//Variable initiation for length of subtitle text, sequence number if its the first line and the subtitle text

		sequence++
		firstrow = true
		subtitle = ""

		//Grab the start time, convert it to a number, then convert the number an SRT valid time string
		sttime = transcription[index].Starttime
		fsttime, err := strconv.ParseFloat(sttime, 64)
		if err != nil {
			fmt.Println(err)
		}
		sttime = Getsrttime(fsttime)

		/*Repeat this until we have either reached the last item in results
		#or the length of the lines we are reading is greater than 64 characters */

		for strlen = 0; (strlen < 64) && (index < len(transcription)); {
			text = transcription[index].Alternatives[0].Content
			strlen += len(text)

			switch classification {

			case "punctuation":
				if len(subtitle) > 0 {
					runes := []rune(subtitle)
					subtitle = string(runes[1 : len(subtitle)-1])
				} else {
					subtitle += text
				}
			default:
				subtitle += (text + " ")
			}

			//If the length of the current string is greater than 32 and this
			//is the first line of the sequence, then add a return character to it

			if (strlen > 32) && (firstrow == true) {
				subtitle += "\n"
				firstrow = false
			}

			/*If the last character is a '.', then we need to set
			the end time attribute to the previous indexes one
			since punctuation characters to not have a time stamp*/

			if classification == "punctuation" {
				entime = transcription[index-1].Endtime
			} else {
				entime = transcription[index].Endtime
			}

			fsttime, err = strconv.ParseFloat(entime, 64)
			entime = Getsrttime(fsttime)

			index++
		}
		//Setup the string that is refers to these two
		//lines in SRT format

		subdetail = fmt.Sprintf("\n%d\n%s --> %s\n%s\n", sequence, sttime, entime, subtitle)

		//Append this to the existing string
		srtinfo += subdetail

	}

	log.Printf(srtinfo)

	return srtinfo, nil
}

//DownloadFile performs the actions of downloading a file
func DownloadFile(filepath string, url string) (bool, error) {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return false, err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return false, err

	}

	return true, nil
}

// Getsrttime - Generates an SRT format time string
func Getsrttime(numerator float64) (timestring string) {

	var h = 3600
	var m = 60
	var s = 1

	integer, frac := math.Modf(numerator)
	integerpart := int(integer)

	hours := integerpart / h
	remainder := integerpart % h

	minutes := remainder / m
	remainder = remainder % m

	seconds := remainder / s
	stringfrac := strconv.FormatFloat(frac, 'f', 3, 64)
	runes := []rune(stringfrac)
	safeSubstring := string(runes[1:len(stringfrac)])

	timestring = fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
	timestring += safeSubstring
	return
}

//Awstranscript - Top level struc for an AWS transcript job output
type Awstranscript struct {
	JobName   string `json:"jobName"`
	Accountid string `json:"accountId"`
	Results   Result `json:"results"`
	Status    string `json:"status"`
}

//Result - Result structure
type Result struct {
	Transcripts []Transcript `json:"transcripts"`
	Items       []Item       `json:"items"`
}

//Transcript - Transcription
type Transcript struct {
	Transcript string `json:"transcript"`
}

//Item - Individual translation word/punctuation record
type Item struct {
	Starttime      string        `json:"start_time"`
	Endtime        string        `json:"end_time"`
	Alternatives   []Alternative `json:"alternatives"`
	Classification string        `json:"type"`
}

// Alternative - Actual translated word and confidence of accuracy
type Alternative struct {
	Confidence string `json:"confidence"`
	Content    string `json:"content"`
}
