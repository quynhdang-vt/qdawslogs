TASK_ID=$1
START_TIME=$2
if [ -z $ENV ]; then
	ENV="prod"
fi
LOGGROUP="/aws/ecs/$ENV-rt"

FILENAME=/tmp/${TASK_ID}.log
#./qdawslogs -messageFilter $TASK_ID -startTime "$START_TIME" -filter "@logStream like /DeepAffects/" > /tmp/${TASK_ID}-DeepAffects.log 2>&1
./qdawslogs -logGroupName ${LOGGROUP} -messageFilter $TASK_ID -startTime "$START_TIME" > ${FILENAME} 2>&1
vi ${FILENAME}

