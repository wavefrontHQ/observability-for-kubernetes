pipeline {
  agent any
  triggers {
    cron '''TZ=America/Denver
44 18 * * 5'''
  }
  stages {
    stage ('Weekly cleanup clusters without keep-me:true label') {
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
      }
      steps {
        script {
          lock("integration-test-gke") {
            sh "scripts/cleanup-gke-clusters.sh"
          }
        }
      }
    }
  }
 }