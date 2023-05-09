pipeline {
  agent any
  options {
    buildDiscarder(logRotator(numToKeepStr: '15'))
  }
  triggers {
    // 4:44PM in Denver Timezone, every Monday to Friday.
    // See: https://www.jenkins.io/doc/book/pipeline/syntax/#cron-syntax
    // MINUTE(0-59) HOUR(0-23) DOM(1-31) MONTH(1-12) DOW(0-7)
    cron '''TZ=America/Denver
44 16 * * 1-5'''
  }
  stages {
    stage('Randomize Team') {
      steps {
        script {
          team_name = '*Team Raven* :disco_raven:'
          team_members = ['Jeremy', 'Jerry', 'Jesse', 'John', 'Peter', 'Yuqi']

          // Prevent the same person from being selected twice in a row.
          (rotating_off, staying_on) = currentBuild.getPreviousBuild().description.tokenize(',')
          team_members -= rotating_off
          Collections.shuffle team_members
          team_members += rotating_off

          currentBuild.description = "${staying_on},${team_members[0]}"
          SLACK_MSG = """
The results are in from <${env.BUILD_URL}|${env.JOB_NAME}> :dice-9823:

${team_name}
${team_members.join('\n')}
"""
          println SLACK_MSG
        }
        slackSend (channel: '#tobs-k8po-team', message: SLACK_MSG)
      }
    }
  }
}
