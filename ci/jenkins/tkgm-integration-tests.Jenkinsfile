pipeline {
  agent {
    label 'nimbus-cloud'
  }

  tools {
    go 'Go 1.20'
  }

  environment {
    PATH = "${env.WORKSPACE}/bin:${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
    VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
    GITHUB_TOKEN = credentials("GITHUB_TOKEN")
    OPERATOR_YAML_TYPE="rc"
  }

  parameters {
    string(defaultValue: "", description: "Git branch of CI pipeline", name: 'GIT_BRANCH_PARAM')
    string(defaultValue: "", description: "Operator YAML SHA of CI pipeline", name: 'OPERATOR_YAML_RC_SHA_PARAM')
  }

  stages {
//     stage ('Find a public pool environment') {
//       steps {
//         script {
          // TODO need more robust logic on whether or not to lock environments as they may fill up quickly
          // sh "scripts/get-tkgm-env-lock.sh 1h"
//           TODO try with 30m lease. Is there a force release? If so, get for an hour and release at the end in a finally block
//         }
//       }
//     }

    stage('Force a failure to test what happens when not waiting') {
      steps {
        sh 'exit 1'
      }
    }

//     stage("Run Collector Integration Tests") {
//       agent {
//         label "worker-5"
//       }
//       options {
//         timeout(time: 18, unit: 'MINUTES')
//       }
//       environment {
//         KUBECONFIG = "$HOME/.kube/config"
//         KUBECONFIG_DIR = "$HOME/.kube"
//         DOCKER_IMAGE = "kubernetes-collector"
//         INTEGRATION_TEST_ARGS="all"
//         INTEGRATION_TEST_BUILD="ci"
//       }
//       steps {
//         lock("integration-test-tkgm") {
//           sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k TKGm'
//           sh 'curl -O http://files.pks.eng.vmware.com/ci/artifacts/shepherd/latest/sheepctl-linux-amd64'
//           sh 'chmod +x sheepctl-linux-amd64 && mv sheepctl-linux-amd64 sheepctl'
//           sh "mkdir -p $KUBECONFIG_DIR"
//
//           sh "./sheepctl -n k8po-team lock list -j | jq -r '. | map(select(.status == \"locked\" and .pool_name != null and (.pool_name | contains(\"tkg\")))) | .[0].access' | jq -r '.tkg[0].kubeconfig' > $KUBECONFIG"
//           sh "chmod go-r $KUBECONFIG"
//           sh 'make clean-cluster'
//           sh 'make -C collector integration-test'
//           sh 'make clean-cluster'
//         }
//       }
//     }

    stage("Run Operator Integration Tests") {
      agent {
        label "worker-5"
      }
      options {
        timeout(time: 18, unit: 'MINUTES')
      }
      environment {
        GCP_CREDS = credentials("GCP_CREDS")
        KUBECONFIG = "$HOME/.kube/config"
        KUBECONFIG_DIR = "$HOME/.kube"
        DOCKER_IMAGE = "kubernetes-operator"
        INTEGRATION_TEST_ARGS = "-r common-metrics -r basic-e2e"
        GIT_BRANCH = "${params.GIT_BRANCH_PARAM}"
        OPERATOR_YAML_RC_SHA = "${params.OPERATOR_YAML_RC_SHA_PARAM}"
      }
      steps {
        lock("integration-test-tkgm") {
          sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh -k TKGm'
          sh 'curl -O http://files.pks.eng.vmware.com/ci/artifacts/shepherd/latest/sheepctl-linux-amd64'
          sh 'chmod +x sheepctl-linux-amd64 && mv sheepctl-linux-amd64 sheepctl'
          sh "mkdir -p $KUBECONFIG_DIR"

          sh "./sheepctl -n k8po-team lock list -j | jq -r '. | map(select(.status == \"locked\" and .pool_name != null and (.pool_name | contains(\"tkg\")))) | .[0].access' | jq -r '.tkg[0].kubeconfig' > $KUBECONFIG"
          sh "chmod go-r $KUBECONFIG"
          sh 'make clean-cluster'
          sh 'make -C operator integration-test'
          sh 'make clean-cluster'
        }
      }
    }
  }
}