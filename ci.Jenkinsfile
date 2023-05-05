pipeline {
  agent {
    label 'nimbus-cloud'
  }

  tools {
    go 'Go 1.20'
  }

  environment {
    PATH = "${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    GITHUB_TOKEN = credentials("GITHUB_TOKEN")
    HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
    PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
    DOCKER_IMAGE = "kubernetes-operator"
    VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
  }

  parameters {
      string(name: 'OPERATOR_YAML_RC_SHA', defaultValue: '')
  }

  stages {
    stage('Test git output') {
      steps {
        sh(returnStdout: true, script: 'git rev-parse --abbrev-ref HEAD').trim()
        sh(returnStdout: true, script: 'git name-rev --name-only HEAD').trim()
        sh(returnStdout: true, script: 'git branch --show-current').trim()
        sh(returnStdout: true, script: 'echo "GIT_BRANCH: ${GIT_BRANCH}"').trim()
        sh(returnStdout: true, script: 'echo "GIT_LOCAL_BRANCH: ${GIT_LOCAL_BRANCH}"').trim()
        sh(returnStdout: true, script: 'echo "GIT_COMMIT: ${GIT_COMMIT}"').trim()
        sh(returnStdout: true, script: 'echo "GIT_PREVIOUS_COMMIT: ${GIT_PREVIOUS_COMMIT}"').trim()
      }
    }
  }
}
