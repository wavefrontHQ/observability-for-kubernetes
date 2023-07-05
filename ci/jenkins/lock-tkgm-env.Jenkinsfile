pipeline {
  agent any
//   options {
//     buildDiscarder(logRotator(numToKeepStr: '15'))
//   }
  triggers {
    // 4:44PM in Denver Timezone, every Monday to Friday.
    // See: https://www.jenkins.io/doc/book/pipeline/syntax/#cron-syntax
    // MINUTE(0-59) HOUR(0-23) DOM(1-31) MONTH(1-12) DOW(0-7)
    cron '''TZ=America/Denver
0 5 * * 1-5'''
  }
  stages {
    stage ('Find a public pool environment') {
      steps {
        script {
          sh "make get-tkgm-lock"
        }
      }
    }
  }
 }