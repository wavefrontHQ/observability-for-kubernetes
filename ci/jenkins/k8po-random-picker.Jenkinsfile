pipeline {
  agent any
  options {
    buildDiscarder(logRotator(numToKeepStr: '5'))
  }
  triggers {
    // MST 4:00 PM (UTC -7) converted to UTC, every Sunday to Thursday.
    // See: https://www.jenkins.io/doc/book/pipeline/syntax/#cron-syntax
    // MINUTE(0-59) HOUR(0-23) DOM(1-31) MONTH(1-12) DOW(0-7)
    cron('0 23 * * 0-4')
  }
  environment {
    REPO_DIR = sh (script: 'git rev-parse --show-toplevel', returnStdout: true).trim()
  }
  stages {
    stage ("Slack message rando-dev results") {
      steps {
        script {
          ORDER_PICKED = sh (script: '$REPO_DIR/scripts/rando-dev.sh', returnStdout: true).trim()
        }
        slackSend (channel: '#tobs-k8po-team', message:
        """
The results are in from <${env.BUILD_URL}|${env.JOB_NAME}>!!!

${ORDER_PICKED}
        """)
      }
    }
  }
}
