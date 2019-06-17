TASK_ID=$1
START_TIME=$2
./qdawslogs -messageFilter $TASK_ID -startTime "$START_TIME" -filter "@logStream like /DeepAffects/" > /tmp/${TASK_ID}-DeepAffects.log 2>&1

