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
    stage ("Slack message rando-dev results for Team Helios") {
      when {
        equals(actual: currentBuild.number % 2, expected: 0)
      }
      environment {
        TEAM_NAME = 'Team Helios :sun_with_face:'
        TEAM_DEV_LIST = 'Anil,Devon,Ginwoo,Glenn,Priya'
      }
      steps {
        script {
          ORDER_PICKED = sh (script: '$REPO_DIR/scripts/rando-dev.sh -n "$TEAM_NAME" -l "$TEAM_DEV_LIST"', returnStdout: true).trim()
        }
        slackSend (channel: '#tobs-k8po-team', message:
        """
The results are in from <${env.BUILD_URL}|${env.JOB_NAME}>!!!

${ORDER_PICKED}
        """)
      }
    }
    stage ("Slack message rando-dev results for Team Raven") {
      when {
        equals(actual: currentBuild.number % 2, expected: 1)
      }
      environment {
        TEAM_NAME = 'Team Raven :raven:'
        TEAM_DEV_LIST = 'Jeremy,Jerry,Jesse,John,Peter,Yuqi'
      }
      steps {
        script {
          ORDER_PICKED = sh (script: '$REPO_DIR/scripts/rando-dev.sh -n "$TEAM_NAME" -l "$TEAM_DEV_LIST"', returnStdout: true).trim()
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
