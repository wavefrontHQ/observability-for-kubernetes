pipeline {
  agent {
    label 'nimbus-cloud'
  }

  tools {
    go 'Go 1.20'
  }

  environment {
    PATH = "${env.WORKSPACE}/bin:${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    GCP_CREDS = credentials("GCP_CREDS")
  }

  triggers {
    cron '''TZ=America/Denver
44 18 * * 5'''
  }

  stages {
    stage ('Weekly cleanup clusters without keep-me:true label') {
      steps {
        script {
          lock("integration-test-gke") {
            sh 'scripts/cleanup-gke-clusters.sh'
          }
        }
      }
    }
  }
 }