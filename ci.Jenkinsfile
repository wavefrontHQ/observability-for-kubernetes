pipeline {
  agent {
    label 'nimbus-cloud'
  }

  stages {
    stage('Trigger collector') {
      steps {
        script {
          def changedFiles = sh returnStdout: true, script: "git show --pretty=\"format:\" --name-only | awk -F\"/\" \"{print \$1}\" | sort -u".split('\n')
          if (changedFiles.any { it =~ /^collector/ }) {
            build job: 'wavefront-collector-for-kubernetes-ci'
          }
        }
      }
    }

    stage('Trigger operator') {
      steps {
        script {
          def changedFiles = sh returnStdout: true, script: "git show --pretty=\"format:\" --name-only | awk -F\"/\" \"{print \$1}\" | sort -u".split('\n')
          if (changedFiles.any { it =~ /^operator/ }) {
            build job: 'wavefront-operator-for-kubernetes-ci'
          }
        }
      }
    }
  }
}
