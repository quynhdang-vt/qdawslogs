TASK_ID=$1
START_TIME=$2
LOG_STREAM=$3
if [ -z $ENV ]; then
	ENV="prod"
fi
LOGGROUP="/aws/ecs/$ENV-rt"

FILENAME=/tmp/${TASK_ID}.log
#./qdawslogs -logGroupName ${LOGGROUP} -messageFilter $TASK_ID -startTime "$START_TIME" -filter "@logStream like /$3/" > ${FILENAME} 2>&1
./qdawslogs -logGroupName ${LOGGROUP} -startTime "$START_TIME" -filter "@logStream like /$3/" > ${FILENAME} 2>&1
vi ${FILENAME}

