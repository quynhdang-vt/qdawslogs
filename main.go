package main

import (
	"flag"
	"fmt"
	"strings"

	"encoding/json"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"log"
	"time"
	"strconv"
	"os"
)

/**
Query cloudwatch log with
See doc:
see aws https://docs.aws.amazon.com/sdk-for-go/api/

https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CWL_QuerySyntax.html

*/

func usage() string {
	return `

      Usage:
        ./qdawslogs [-logGroupName xxx] [-field xx]* -filter FILTER_CLAUSE or -messageFilter [-startTime epoch/RFC3339] [-endTime epoch/RFC3339] [-limit xxx] [-region xxx]

      Required:  -filter or -messageFilter. 

                 Filter must be a complete filter clause.  See https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CWL_QuerySyntax.html
                 MessageFilter is the value to be included in the like clause.

      -field is optional.  Can be specified multiple times.  When specified, it should be one of the following:  @timestamp, @message, @logStream or @ingestionTime

      -startTime/-endTime can be either an integer for the epoch time in seconds or RFC 3339 format (which is the case of graphQL datetime value

      Optional with provided values:
        logGroupName = /aws/ecs/prod-rt
        region=us-east-1
        field = @timestamp, @message, @logStream
        startTime = 1hour before now
        endTime = now


      -------------------
      Example:

        [1]  Getting fields timestamp, message from the log group /aws/ecs/stage-rt within the previous hour of the given epoch time: 

		       ./qdawslogs -logGroupName /aws/ecs/stage-rt -field @timestamp -field @message -filter "@message like /19062412_5Xi2eYcEc6/" -endTime 1560322977 -limit 1000
      



        [2] Getting default fields (timestamp, message, logStream) from the default log group /aws/ecs/prod-rt with startTime 2019-06-12T06:47:12.000Z, filtering 
            messages containing 19062412_5Xi2eYcEc6:

		        ./qdawslogs.mac -startTime "2019-06-12T06:47:12.000Z" -messageFilter 19062412_5Xi2eYcEc6




        [3] Getting default fields (timestamp, message, logStream) from the default log group /aws/ecs/prod-rt with startTime 2019-06-12T06:47:12.000Z, filtering 
            messages containing 19062412_5Xi2eYcEc6 AND logStream contains "coord":

                ./qdawslogs.mac -startTime "2019-06-12T06:47:12.000Z" -messageFilter 19062412_5Xi2eYcEc6 -filter "@logStream like /coord/"
 

`
}

func getTimestampFromRFC3339(tString string) int64 {
	if tString == "" {
		return 0
	}
	res, err := time.Parse(time.RFC3339, tString)
	if err == nil {
		return res.Unix()
	}
	return 0
}

func main() {
	startQueryInput, region := parseArguments()
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(region),
	}))
	cwLogs := cloudwatchlogs.New(sess)

	startQueryOutput, err := cwLogs.StartQuery(&startQueryInput)
	if err != nil {
		log.Fatalf("\n\nERROR Failed to Start Query, err=%v\n", err)
	}

	var getQueryResultsInput = cloudwatchlogs.GetQueryResultsInput{
		QueryId: startQueryOutput.QueryId,
	}

	done := false
	for !done {
		log.Printf("... Waiting to retrieve results... \n")
		time.Sleep(10 * time.Second)
		getQueryResultsOutput, err := cwLogs.GetQueryResults(&getQueryResultsInput)

		log.Printf(">>>>>> GetQueryResults= %d records, stats=%s, status=%s\n", len(getQueryResultsOutput.Results),
			ToString(getQueryResultsOutput.Statistics),  *getQueryResultsOutput.Status)
		fmt.Println("----------------")
		if err != nil {
			log.Printf("GetQueryResults got err=%v\n", err)
			done = true
		} else {

			switch *getQueryResultsOutput.Status {
			case "Complete":
				done = true
				fallthrough
			default:
				for _, fieldResults := range getQueryResultsOutput.Results {
					var buf strings.Builder
					for _, fields := range fieldResults {
						if strings.Contains(*fields.Field, "@ptr") {
							continue
						}
						buf.WriteString(*fields.Field)
						buf.WriteString("=")
						buf.WriteString(*fields.Value)
						buf.WriteString(", ")
					}
					fmt.Printf("%s\n\n", buf.String())
				}

			}
		}

	}
}

type flagArgs []string

func (a flagArgs) String() string {
	return strings.Join(a.Args(), ",")
}

func (a *flagArgs) Set(value string) error {
	*a = append(*a, value)
	return nil
}
func (a flagArgs) Args() []string {
	return []string(a)
}

func ToString(c interface{}) string {
	s, _ := json.MarshalIndent(c, "", "\t")
	return string(s)
}

func parseArguments() (input cloudwatchlogs.StartQueryInput, region string) {
	var logGroupName string
	var messageFilter string
	var fields []string
	var filters []string
	var startTime int
	var endTime int

	var limit int64
	var argRegion string

	var fieldArgs, filterArgs flagArgs

	var argStartTime string
	var argEndTime string

	flag.StringVar(&logGroupName, "logGroupName", "/aws/ecs/prod-rt", "specify log group name")
	flag.StringVar(&argRegion, "region", "us-east-1", "specify AWS region")

	flag.Var(&fieldArgs, "field", "return field names, ampersand is required, e.g. @timestamp, @message")
	flag.Var(&filterArgs, "filter", "filters, e.g. \"@message like /xxx/\"")

	flag.StringVar(&argStartTime, "startTime", "", "specify startTime (epoch in sec) for query scope")
	flag.StringVar(&argEndTime, "endTime", "", "specify endTime (epoch in sec) for query scope")
	flag.Int64Var(&limit, "limit", 0, "specify limit (epoch in sec) for query scope")
	flag.StringVar(&messageFilter, "messageFilter", "", "filter for @message, the vaue for like op to the @message field, typically jobId or taskId")

	flag.Parse()

	if flag.NFlag() != 0 {
		fields = append([]string{}, fieldArgs.Args()...)
		filters = append([]string{}, filterArgs.Args()...)
	} else {
		fmt.Println(usage())
		os.Exit(1)
	}

	if len(fields) == 0 {
		fields = []string{"@timestamp", "@logStream", "@message"}
	} else {
		invalidFields := []string{}
		timestampSeen := false
		for _, field := range fields {
			// check to make sure that they are in @timestamp, @logStream, @message or
			if field != "@message" && field != "@timestamp" && field != "logStream" && field != "@ingestionTime" {
				invalidFields = append(invalidFields, field)
			}
			if field == "@timestamp" {
				timestampSeen = true
			}
		}
		if !timestampSeen {
			fields = append(fields, "@timestamp")
		}
		if len(invalidFields) > 0 {
			log.Fatalf("\n\nERROR Invalid fields = %s\n%s\n", ToString(invalidFields), usage())
		}
	}

	if len(filters) == 0 {
		if len(messageFilter) == 0 {
			log.Fatalf("\n\nERROR Missing filter or messageFilter, for example -filter=\"@messsage like /19023434_xxx/\" or -messageFilter=1992344_9sf34\n%s", usage())
		}
	}
	if len(messageFilter) != 0 {
		filters = append(filters, fmt.Sprintf("@message like /%s/", messageFilter))
	}

	var err error
	if argEndTime!="" {
		endTime, err = strconv.Atoi(argEndTime)
		if err != nil {
			endTime = int(getTimestampFromRFC3339(argEndTime))
		}
	}

	if endTime == 0 {
		endTime = int(time.Now().Unix())
	}

	if argStartTime!="" {
		startTime, err = strconv.Atoi(argStartTime)
		if err != nil {
			startTime = int(getTimestampFromRFC3339(argStartTime))
		}
	}

	// an hour from endTime if not given
	if startTime == 0 {
		startTime = endTime - 3600
	}

	var str strings.Builder

	str.WriteString("fields ")
	for i, field := range fields {
		str.WriteString(field)
		if i < len(fields)-1 {
			str.WriteString(", ")
		}
	}
	str.WriteString(" | sort @timestamp desc ")
	for _, filter := range filters {
		str.WriteString(fmt.Sprintf("| filter %s", filter))
	}
	input = cloudwatchlogs.StartQueryInput{}
	input.SetLogGroupName(logGroupName)
	input.SetStartTime(int64(startTime))
	input.SetEndTime(int64(endTime))
	if limit > 0 {
		input.SetLimit(limit)
	}
	input.SetQueryString(str.String())
	log.Printf("StartQueryInput=%s\n", input.String())
	err = input.Validate()
	if err != nil {
		log.Fatalf("\n\nERROR in formulating CloudWatch StartQueryInput, err=%v", err)
	}
	return input, argRegion
}
