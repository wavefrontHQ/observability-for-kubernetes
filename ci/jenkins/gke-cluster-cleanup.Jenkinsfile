pipeline {
  agent any

  tools {
    go 'Go 1.20'
  }

  environment {
    PATH = "${env.WORKSPACE}/bin:${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    GCP_CREDS = credentials("GCP_CREDS")
  }

  triggers {
    cron '0 */24 * * *'
  }

  stages {
    stage ('Cleanup GKE clusters') {
      steps {
        sh 'scripts/cleanup-gke-clusters.sh'
      }
    }
  }
}
