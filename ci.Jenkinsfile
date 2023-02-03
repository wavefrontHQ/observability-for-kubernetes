pipeline {
  stages {
    stage("Regression") {
    }
    stage("Abort") {
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
