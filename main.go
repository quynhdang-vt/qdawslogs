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
        ./qdawslogs [-logGroupName xxx] [-field xx]* -filter FILTER_CLAUSE or -messageFilter [-startTime epoch] [-endTime epoch] [-limit xxx] [-region xxx]
      Example:
		./qdawslogs -logGroupName /aws/ecs/stage-rt -field @timestamp -field @message -filter "@message like /xxx/" -startTime #### -endTime #### -limit 1000
 
      Required:  -filter or -messageFilter. 

                 Filter must be a complete filter clause.  See https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CWL_QuerySyntax.html
                 MessageFilter is the value to be included in the like clause.

      -field is optional.  Can be specified multiple times.  When specified, it should be one of the following:  @timestamp, @message, @logStream or @ingestionTime

      Optional with provided values:
        logGroupName = /aws/ecs/prod-rt
        region=us-east-1
        field = @timestamp, @message, @logStream
        startTime = 1hour before now
        endTime = now
`
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
		getQueryResultsOutput, err := cwLogs.GetQueryResults(&getQueryResultsInput)
		if err != nil {
			log.Printf("GetQueryResults got err=%v\n", err)
			done = true
		} else {

			switch *getQueryResultsOutput.Status {
			case "Running":
				fallthrough
			case "Scheduled":
				time.Sleep(5 * time.Second)
			case "Complete":
				done = true
				fallthrough
			default:
				for _, fieldResults := range getQueryResultsOutput.Results {
					var buf strings.Builder
					for _, fields := range fieldResults {
						buf.WriteString(*fields.Field)
						buf.WriteString("=")
						buf.WriteString(*fields.Value)
						buf.WriteString(", ")
					}
					log.Printf("%s\n", buf.String())
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
	var startTime int64
	var endTime int64
	var limit int64
	var argRegion string

	var fieldArgs, filterArgs flagArgs

	flag.StringVar(&logGroupName, "logGroupName", "/aws/ecs/prod-rt", "specify log group name")
	flag.StringVar(&argRegion, "region", "us-east-1", "specify AWS region")

	flag.Var(&fieldArgs, "field", "return field names, ampersand is required, e.g. @timestamp, @message")
	flag.Var(&filterArgs, "filter", "filters, e.g. \"@message like /xxx/\"")

	flag.Int64Var(&startTime, "startTime", 0, "specify startTime (epoch in sec) for query scope")
	flag.Int64Var(&endTime, "endTime", 0, "specify endTime (epoch in sec) for query scope")
	flag.Int64Var(&limit, "limit", 0, "specify limit (epoch in sec) for query scope")
	flag.StringVar(&messageFilter, "messageFilter", "", "filter for @message, the vaue for like op to the @message field, typically jobId or taskId")

	flag.Parse()

	if flag.NFlag() != 0 {
		fields = append([]string{}, fieldArgs.Args()...)
		filters = append([]string{}, filterArgs.Args()...)
	} else {
		log.Fatal(usage())
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

	if endTime == 0 {
		endTime = time.Now().Unix()
	}
	// an hour from endTime
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
	input.SetStartTime(startTime)
	input.SetEndTime(endTime)
	if limit > 0 {
		input.SetLimit(limit)
	}
	input.SetQueryString(str.String())
	log.Printf("StartQueryInput=%s\n", input.String())
	err := input.Validate()
	if err != nil {
		log.Fatalf("\n\nERROR in formulating CloudWatch StartQueryInput, err=%v", err)
	}
	return input, argRegion
}
