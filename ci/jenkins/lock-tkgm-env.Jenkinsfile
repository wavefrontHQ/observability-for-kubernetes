pipeline {
  agent any
//   options {
//     buildDiscarder(logRotator(numToKeepStr: '15'))
//   }
//   triggers {
//     // 4:44PM in Denver Timezone, every Monday to Friday.
//     // See: https://www.jenkins.io/doc/book/pipeline/syntax/#cron-syntax
//     // MINUTE(0-59) HOUR(0-23) DOM(1-31) MONTH(1-12) DOW(0-7)
//     cron '''TZ=America/Denver
// 44 16 * * 1-5'''
//   }
  stages {

    stage ('Find a public pool environment') {
      steps {
            script {
              sh 'curl -O http://files.pks.eng.vmware.com/ci/artifacts/shepherd/latest/sheepctl-linux-amd64'
              sh 'chmod +x sheepctl-linux-amd64 && mv sheepctl-linux-amd64 sheepctl'
              sh './sheepctl pool list --public -u shepherd.run'
              sh './sheepctl target set -u shepherd.run -n k8po-team'
//               sh 'sheepctl pool lock tkg-2.1-vcenter-7.0.0 --from-namespace shepherd-official --lifetime 1h --output lock.json'
            }
      }
    }
  }
 }