pipeline {
  stages {
    stage("Aborted") {
      steps {
        sh 'sleep 4000'
      }
    }
    stage("Regression") {
      steps {
        sh 'exit 1'
      }
    }
    stage("Fixed") {
      steps {
        sh 'exit 0'
      }
    }
  }

  post {
    aborted {
      slackSend (channel: '#open-channel', color: '#FF0000', message: "aborted")
    }
    regression {
      slackSend (channel: '#open-channel', color: '#FF0000', message: "regression")
    }
    fixed {
      slackSend (channel: '#open-channel', color: '#008000', message: "fixed")
    }
  }
}
