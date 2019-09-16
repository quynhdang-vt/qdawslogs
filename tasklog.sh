# Usage
#
# ENV={stage,prod} ./tasklog.sh {TASK_ID} {createdDateTime}
#
# Default ENV is prod.
#
# Note that this uses the default AWS Credentials
# TO specify a different profile, set the AWS_PROFILE environment variable
#
TASK_ID=$1
START_TIME=$2
if [ -z $ENV ]; then
	ENV="prod"
fi
LOGGROUP="/aws/ecs/$ENV-rt"

FILENAME=/tmp/${TASK_ID}.log
#osx/qdawslogs -messageFilter $TASK_ID -startTime "$START_TIME" -filter "@logStream like /DeepAffects/" > /tmp/${TASK_ID}-DeepAffects.log 2>&1
CMD=qdawslogs

if [ "$(uname)" == "Darwin" ]; then
  CMD=osx/qdawslogs
else
  CMD=linux/qdawslogs
fi

echo "Running CW query .. will pipe to ${FILENAME}"
set -x
${CMD} -logGroupName ${LOGGROUP} -limit 10000 -messageFilter $TASK_ID -startTime "$START_TIME" > ${FILENAME} 2>&1
set +x
cat ${FILENAME}

