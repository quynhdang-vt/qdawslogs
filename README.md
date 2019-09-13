# qdawslogs

This CLI tool is similar to Cloudwatch Insights by submitting queries to Cloudwatch and retrieves log messages for a log group.



## Building

Make sure you have Go, make then just 

```
make build
```

Depending on your env, it will go to either osx or linux.
Note that I have pre-built both versions and checked in here.


## Running

### AWS credentials

Make sure that you have configured AWS credentials in a file `~/.aws/credentials` or via environment variables
```
AWS_ACCESS_KEY_ID=xxx
AWS_SECRET_ACCESS_KEY=xxxx
```


### Usage

Running the tool without any parameters will provide the usage instructions and samples

```


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


```


The output can then be piped to a file, e.g.

```
./qdawslogs.mac -startTime "2019-06-12T06:47:12.000Z" -messageFilter 19062412_5Xi2eYcEc6 -filter "@logStream like /coord/" > samples/coord-19062412_5Xi2eYcEc6.log 2>&1
```

OR

```
./qdawslogs -startTime "2019-06-12T06:47:12.000Z" -messageFilter 19062412_5Xi2eYcEc6 > samples/all=19062412_5Xi2eYcEc6.log 2>&1
```


OR

```
./qdawslogs -startTime "2019-06-12T06:47:12.000Z" -messageFilter 19062412_5Xi2eYcEc6 -filter "@logStream like /DeepAffects/" > samples/DeepAffects-19062412_5Xi2eYcEc6.log 2>&1
```

### Sample outputs

See https://github.com/quynhdang-vt/qdawslogs/tree/master/samples for the above sample outputs

### Helpful scripts

#### tasklog.sh

Use this tool to query CW for messages related to a job/task for an ECS env, e.g. `dev`, `stage`, or `prod`.

For example:
To get log messages that have the taskId / jobId in the message from a given startTime (hint: use graphql to query for the createdDateTime` -- you can have it either as a UTC string format or epoch

```
ENV=dev ./tasklog.sh 19062625_v0BTnn4F0Z 1561480000
```

#### Getting logstream of an engine sample:

For Face Detection on Stage -- the logStream can be found in the taskLog script above.

```
./qdawslogs -logGroupName /aws/ecs/stage-rt -startTime 1562580145 -filter "@logStream like /FaceDetectionEV2F_E/" -filter "@logStream like /4b5234ef-437d-410f-a1ae-4ef26bac1db3/" > /tmp/19072808_UVMJrAX7UTORD0Q-logstream.txt 2>&1
```

