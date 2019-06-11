package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"log"
	"time"
	"github.com/aws/aws-sdk-go/aws/session"
)

/**
Query cloudwatch log with
See doc:
see aws https://docs.aws.amazon.com/sdk-for-go/api/

https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/CWL_QuerySyntax.html


Usage

-filterMsg xxxx -startTime epochInSec -endTime epochInSec

*/
func main() {

	startQueryInput := parseArguments()
	sess := session.Must(session.NewSession())
	cwLogs := cloudwatchlogs.New(sess)

	startQueryOutput, err:=cwLogs.StartQuery(&startQueryInput)
	if err!=nil {
		log.Fatalf("Failed to Start Query, err=%v\n", err)
	}

	var getQueryResultsInput = cloudwatchlogs.GetQueryResultsInput{
		QueryId: startQueryOutput.QueryId,
	}

	done:=false
	for !done {
		getQueryResultsOutput, err:=cwLogs.GetQueryResults(&getQueryResultsInput)
		if err!=nil {
			log.Printf("GetQueryResults got err=%v\n", err)
			done=true
		} else {

			switch *getQueryResultsOutput.Status {
			case "Running":
				fallthrough
			case "Scheduled":
				time.Sleep(5 * time.Second)
			case "Complete":
				done=true
				fallthrough
			default:
				for _, fieldResults:=range getQueryResultsOutput.Results {
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

/*

	if len(states) == 0 {
		fmt.Fprintf(os.Stderr, "error: %v\n", usage())
		os.Exit(1)
	}
	instanceCriteria := " "
	for _, state := range states {
		instanceCriteria += "[" + state + "]"
	}

	if len(regions) == 0 {
		var err error
		regions, err = fetchRegion()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
	}

	for _, region := range regions {
		sess := session.Must(session.NewSession(&aws.Config{
			Region: aws.String(region),
		}))

		ec2Svc := ec2.New(sess)
		params := &ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("instance-state-name"),
					Values: aws.StringSlice(states),
				},
			},
		}

		result, err := ec2Svc.DescribeInstances(params)
		if err != nil {
			fmt.Println("Error", err)
		} else {
			fmt.Printf("\n\n\nFetching instance details for region: %s with criteria: %s**\n ", region, instanceCriteria)
			if len(result.Reservations) == 0 {
				fmt.Printf("There is no instance for the region: %s with the matching criteria:%s  \n", region, instanceCriteria)
			}
			for _, reservation := range result.Reservations {

				fmt.Println("printing instance details.....")
				for _, instance := range reservation.Instances {
					fmt.Println("instance id " + *instance.InstanceId)
					fmt.Println("current State " + *instance.State.Name)
				}
			}
			fmt.Printf("done for region %s **** \n", region)
		}
	}
}

func fetchRegion() ([]string, error) {
	awsSession := session.Must(session.NewSession(&aws.Config{}))

	svc := ec2.New(awsSession)
	awsRegions, err := svc.DescribeRegions(&ec2.DescribeRegionsInput{})
	if err != nil {
		return nil, err
	}

	regions := make([]string, 0, len(awsRegions.Regions))
	for _, region := range awsRegions.Regions {
		regions = append(regions, *region.RegionName)
	}

	return regions, nil
}
*/

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

func parseArguments() (input cloudwatchlogs.StartQueryInput) {
	var logGroupName string
	var messageFilter string
	var fields []string
	var filters []string
	var startTime int64
	var endTime int64
	var limit int64

	var fieldArgs, filterArgs flagArgs

	flag.StringVar(&logGroupName, "logGroupName", "/aws/ecs/prod-rt", "specify log group name")
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
		fields = []string{"@timestamp", "@message", "@logStream"}
	}
	if len(filters) == 0 {
		if len(messageFilter) == 0 {
			log.Fatalf("Missing filter or messageFilter, for example -filter=\"@messsage like /19023434_xxx/\" or -messageFilter=1992344_9sf34\n%s", usage())
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
		log.Fatalf("Error in formulating CloudWatch StartQueryInput, err=%v", err)
	}
	return input
}

func usage() string {
	return `
      Example:
		./qdawslogs -logGroupName -field fieldA -field fieldB -filter "@message like /xxx/" -startTime #### -endTime #### -limit 1000
 
      Required:  filter or messageFilter.  
                 Filter must be complete.  MessageFilter is the value to be included in the like clause.

      Optional with provided values:
        logGroupName = /aws/ecs/prod-rt
        field = @timestamp, @message, @logStream
        startTime = 1hour before now
        endTime = now
`
}
