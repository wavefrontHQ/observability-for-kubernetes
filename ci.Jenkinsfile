pipeline {
  agent {
    label 'nimbus-cloud'
  }

  tools {
    go 'Go 1.20'
  }

  environment {
    PATH = "${env.WORKSPACE}/bin:${env.HOME}/go/bin:${env.HOME}/google-cloud-sdk/bin:${env.PATH}"
    GITHUB_TOKEN = credentials("GITHUB_TOKEN")
    HARBOR_CREDS = credentials("projects-registry-vmware-tanzu_observability_keights_saas-robot")
    PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
    VERSION_POSTFIX = "-alpha-${GIT_COMMIT.substring(0, 8)}"
    WAVEFRONT_TOKEN = credentials("WAVEFRONT_TOKEN_NIMBA")
  }

  parameters {
    string(name: 'OPERATOR_YAML_RC_SHA', defaultValue: '')
    string(name: 'METRICS_RETRY_COUNT', defaultValue: '50')
    booleanParam(
      name: 'FORCE_RUN_CI',
      defaultValue: false,
      description: 'To manually trigger a build, activate the FORCE_RUN_CI checkbox.'
    )
  }

  stages {
    stage("Set RUN_CI") {
      environment {
        FILES_TO_CHECK = 'operator scripts collector ci.Jenkinsfile Makefile'
      }
      steps {
        script {
          if (params.FORCE_RUN_CI) {
            env.RUN_CI = 'true'
          } else if (env.BRANCH_NAME == 'main') {
            env.RUN_CI = sh(script: './ci/jenkins/run-ci.sh -b ${GIT_COMMIT}~ -d ${GIT_COMMIT} -f "${FILES_TO_CHECK}"', returnStdout: true).trim()
          } else {
            env.RUN_CI = sh(script: './ci/jenkins/run-ci.sh -b origin/main -d ${GIT_COMMIT} -f "${FILES_TO_CHECK}"', returnStdout: true).trim()
          }
        }
      }
    }

    stage("Go Tests and Publish Images") {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      parallel{
        stage("Publish Collector") {
          agent {
            label "worker-1"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            RELEASE_TYPE = "alpha"
            DOCKER_IMAGE = "kubernetes-collector"
          }
          steps {
             sh 'cd collector && ./hack/jenkins/install_docker_buildx.sh'
             sh 'cd collector'
             sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
             sh 'cd collector && HARBOR_CREDS_USR=$(echo $HARBOR_CREDS_USR | sed \'s/\\$/\\$\\$/\') make clean docker-xplatform-build'
          }
        }

        stage("Publish Operator") {
          agent {
            label "worker-2"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            RELEASE_TYPE = "alpha"
            COLLECTOR_PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            TOKEN = credentials('GITHUB_TOKEN')
            COLLECTOR_IMAGE = "kubernetes-collector"
          }
          steps {
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
            sh 'cd operator && make clean-build'
            sh 'cd operator && echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
            sh 'cd operator && make docker-xplatform-build'
            sh 'cd operator && ./hack/jenkins/create-rc-ci.sh'
            script {
              env.OPERATOR_YAML_RC_SHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
            }
          }
        }

        stage("Collector Go Tests") {
          agent {
            label "worker-3"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          steps {
            sh 'cd collector && make checkfmt vet tests'
          }
        }
        stage("Operator Go Tests") {
          agent {
            label "worker-4"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          steps {
            sh 'cd operator && make checkfmt vet test'
            sh 'cd operator && make golangci-lint'
          }
        }

        stage("Test Openshift build") {
          agent {
            label "worker-5"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          steps {
            sh 'cd collector && docker build -f deploy/docker/Dockerfile-rhel .'
          }
        }
      }
    }

    stage('Run Integration Tests') {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      environment {
        METRICS_RETRY_COUNT = "${params.METRICS_RETRY_COUNT}"
        TOKEN = credentials('GITHUB_TOKEN')
      }
      // To save time, the integration tests and wavefront-metrics tests are split up between gke and eks
      // But we want to make sure that the combined and default integration tests are run on both
      parallel {
        stage("Kind K8s v1.21.14") {
          agent {
            label "worker-kind-k8s-1-21"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            DOCKER_IMAGE = "kubernetes-collector"
            INTEGRATION_TEST_ARGS="all"
            INTEGRATION_TEST_BUILD="ci"
            KIND_K8S_VERSION='v1.21.14'
          }
          steps {
            lock("integration-test-kind-1-21-14") {
              sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k kind'
              sh './scripts/get-or-create-kind-cluster.sh'

              sh 'cd collector && make clean-cluster'
              sh 'cd collector && make integration-test'
              sh 'cd collector && make clean-cluster'
            }
          }
        }

        stage("Kind K8s v1.22.17") {
          agent {
            label "worker-kind-k8s-1-22"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            DOCKER_IMAGE = "kubernetes-collector"
            INTEGRATION_TEST_ARGS="all"
            INTEGRATION_TEST_BUILD="ci"
            KIND_K8S_VERSION='v1.22.17'
          }
          steps {
            lock("integration-test-kind-1-22-17") {
              sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k kind'
              sh './scripts/get-or-create-kind-cluster.sh'

              sh 'cd collector && make clean-cluster'
              sh 'cd collector && make integration-test'
              sh 'cd collector && make clean-cluster'
            }
          }
        }

        stage("Kind K8s v1.23.17") {
          agent {
            label "worker-kind-k8s-1-23"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            DOCKER_IMAGE = "kubernetes-collector"
            INTEGRATION_TEST_ARGS="all"
            INTEGRATION_TEST_BUILD="ci"
            KIND_K8S_VERSION='v1.23.17'
          }
          steps {
            lock("integration-test-kind-1-23-17") {
              sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k kind'
              sh './scripts/get-or-create-kind-cluster.sh'

              sh 'cd collector && make clean-cluster'
              sh 'cd collector && make integration-test'
              sh 'cd collector && make clean-cluster'
            }
          }
        }

//         stage("GKE") {
//           agent {
//             label "worker-1"
//           }
//           options {
//             timeout(time: 60, unit: 'MINUTES')
//           }
//           environment {
//             DOCKER_IMAGE = "kubernetes-operator"
//             OPERATOR_YAML_TYPE="rc"
//             GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
//             GCP_ZONE="a"
//             GCP_CREDS = credentials("GCP_CREDS")
//             GCP_PROJECT = "wavefront-gcp-dev"
//           }
//           steps {
//             sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
//             sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
//             lock("integration-test-gke") {
//               sh 'cd operator && make gke-connect-to-cluster'
//               sh 'make clean-cluster'
//               sh 'make -C operator integration-test'
//               sh 'make clean-cluster'
//             }
//           }
//         }
//
//         stage("EKS") {
//           agent {
//             label "worker-3"
//           }
//           options {
//             timeout(time: 60, unit: 'MINUTES')
//           }
//           environment {
//             DOCKER_IMAGE = "kubernetes-operator"
//             OPERATOR_YAML_TYPE="rc"
//             GCP_CREDS = credentials("GCP_CREDS")
//             AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
//             AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
//             INTEGRATION_TEST_ARGS="-r advanced -r common-metrics"
//           }
//           steps {
//             sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
//             sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
//             lock("integration-test-eks") {
//               sh 'cd operator && make target-eks'
//               sh 'make clean-cluster'
//               sh 'make -C operator integration-test'
//               sh 'make clean-cluster'
//             }
//           }
//         }
//
//         stage("AKS") {
//           agent { label "worker-4" }
//           options { timeout(time: 60, unit: 'MINUTES') }
//           environment {
//             DOCKER_IMAGE = "kubernetes-operator"
//             OPERATOR_YAML_TYPE="rc"
//             GCP_CREDS = credentials("GCP_CREDS")
//             AKS_CLUSTER_NAME = "k8po-ci"
//             INTEGRATION_TEST_ARGS = '-r validation-errors -r validation-legacy -r validation-errors-preprocessor-rules -r allow-legacy-install -r common-metrics'
//           }
//           steps {
//             sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
//             sh 'cd operator && ./hack/jenkins/install_docker_buildx.sh'
//             lock("integration-test-aks") {
//               withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
//                 sh 'kubectl config use k8po-ci'
//                 sh 'make clean-cluster'
//                 sh 'make -C operator integration-test'
//                 sh 'make clean-cluster'
//               }
//             }
//           }
//         }
      }
    }
  }

  post {
    failure {
      script {
        if (env.BRANCH_NAME == 'main') {
          slackSend (channel: '#tobs-k8po-team', color: '#FF0000', message: "CI BUILD FAILED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
        }
      }
    }
    fixed {
      script {
        if (env.BRANCH_NAME == 'main') {
          slackSend (channel: '#tobs-k8po-team', color: '#008000', message: "CI BUILD FIXED: <${env.BUILD_URL}|${env.JOB_NAME} [${env.BUILD_NUMBER}]>")
        }
      }
    }
  }
}
