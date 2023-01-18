pipeline {
  agent {
    label 'nimbus-cloud'
  }

  stages {
    stage('Trigger collector') {
      steps {
        script {
          def collectorChangesStatusCode = sh returnStatus: true, script: "[[ \"${$(git show --pretty=\"format:\" --name-only | awk -F\"/\" \"{print $1}\" | sort -u)[*]}\" =~ \"collector\" ]]"
          if (!collectorChangesStatusCode) {
            build job: 'wavefront-collector-for-kubernetes-ci'
          }
        }
      }
    }

    stage('Trigger operator') {
      steps {
        script {
          def operatorChangesStatusCode = sh returnStatus: true, script: "[[ \"${$(git show --pretty=\"format:\" --name-only | awk -F\"/\" \"{print $1}\" | sort -u)[*]}\" =~ \"operator\" ]]"
          if (!operatorChangesStatusCode) {
            build job: 'wavefront-operator-for-kubernetes-ci'
          }
        }
      }
    }
  }
}
