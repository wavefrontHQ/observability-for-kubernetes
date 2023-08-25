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
    stage('Randomize Team Helios') {
      steps {
        script {
          team_name = '*Team Helios* :awesome_sun:'
          team_members = ['Devon', 'Ginwoo', 'Glenn', 'Kyle', 'Mark', 'Matt']
          if (Calendar.THURSDAY == Calendar.getInstance(TimeZone.getTimeZone("America/Denver")).get(Calendar.DAY_OF_WEEK)) {
            team_members -= 'Devon'
          }

          // Prevent the same person from being selected twice in a row.
          previous_build_description = currentBuild.getPreviousBuild().description
          if (currentBuild.getPreviousBuild().description) {
            rotating_off_list = previous_build_description.split(',')
            rotating_off = rotating_off_list[0]
          } else {
            rotating_off = ''
          }

          team_members -= rotating_off
          Collections.shuffle team_members
          team_members += rotating_off

          currentBuild.description = "${team_members[0]}"
          SLACK_MSG = """
The results are in from <${env.BUILD_URL}|${env.JOB_NAME}> :dice-9823:

${team_name}
${team_members.join('\n')}
"""
          println SLACK_MSG
        }
        slackSend (channel: '#cpo-team-helios', message: SLACK_MSG)
      }
    }
    stage('Randomize Team Raven') {
      steps {
        script {
          team_name = '*Team Raven* :disco_raven:'
          team_members = ['Anil', 'Jerry', 'John', 'Yuqi'] // 'Jeremy' is on paternity leave

          // Prevent the same person from being selected twice in a row.
          previous_build_description = currentBuild.getPreviousBuild().description
          if (currentBuild.getPreviousBuild().description) {
            rotating_off_list = previous_build_description.split(',')
            rotating_off = rotating_off_list[1]
          } else {
            rotating_off = ''
          }

          team_members -= rotating_off
          Collections.shuffle team_members
          team_members += rotating_off

          currentBuild.description += ",${team_members[0]}"
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
