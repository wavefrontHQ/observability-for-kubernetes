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
    DOCKER_IMAGE = "kubernetes-operator"
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
            sh 'cd collector && docker build -f hack/deploy/docker/Dockerfile-rhel .'
          }
        }
      }
    }

    stage('Run Integration Tests') {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      environment {
        OPERATOR_YAML_TYPE="rc"
        TOKEN = credentials('GITHUB_TOKEN')
        METRICS_RETRY_COUNT = "${params.METRICS_RETRY_COUNT}"
      }
      // To save time, the integration tests and wavefront-metrics tests are split up between gke and eks
      // But we want to make sure that the combined and default integration tests are run on both
      parallel {
        stage("GKE Collector") {
          agent {
            label "gke-integration-worker"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
            GCP_ZONE="a"
            DOCKER_IMAGE = "kubernetes-collector"
            INTEGRATION_TEST_ARGS="all"
            INTEGRATION_TEST_BUILD="ci"
          }
          steps {
            /* Setup */
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'make gke-connect-to-cluster'

            lock("integration-test-gke-collector") {
              /* Collector Integration Tests */
              sh 'make clean-cluster'
              sh 'make -C collector integration-test'
              sh 'CLEAN_CLUSTER_ARGS="-n" make clean-cluster'
            }
          }
        }

        stage("GKE Operator 1") {
          agent {
            label "gke-operator-worker-1"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-operator-1"
            GCP_ZONE="a"
            INTEGRATION_TEST_ARGS="-r k8s-events-only -r validation-errors -r validation-legacy -r validation-errors-preprocessor-rules -r basic"
          }
          steps {
            /* Setup */
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'

            lock("integration-test-gke-operator-1") {
              sh './ci/jenkins/get-or-create-cluster.sh'

              /* Operator Integration Tests */
              sh 'make clean-cluster'
              sh 'make -C operator integration-test'
              sh 'CLEAN_CLUSTER_ARGS="-n" make clean-cluster'
            }
          }
        }

        stage("GKE Operator 2") {
          agent {
            label "gke-operator-worker-2"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-operator-2"
            GCP_ZONE="a"
            INTEGRATION_TEST_ARGS="-r logging-integration -r allow-legacy-install -r common-metrics"
          }
          steps {
            /* Setup */
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'

            lock("integration-test-gke-operator-2") {
              sh './ci/jenkins/get-or-create-cluster.sh'

              /* Operator Integration Tests */
              sh 'make clean-cluster'
              sh 'make -C operator integration-test'
              sh 'CLEAN_CLUSTER_ARGS="-n" make clean-cluster'
            }
          }
        }

        stage("GKE Operator 3") {
          agent {
            label "gke-operator-worker-3"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GCP_PROJECT = "wavefront-gcp-dev"
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-operator-3"
            GCP_ZONE="a"
            INTEGRATION_TEST_ARGS="-r advanced -r with-http-proxy -r control-plane"
          }
          steps {
            /* Setup */
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k gke'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'

            lock("integration-test-gke-operator-3") {
              sh './ci/jenkins/get-or-create-cluster.sh'

              /* Operator Integration Tests */
              sh 'make clean-cluster'
              sh 'make -C operator integration-test'
              sh 'CLEAN_CLUSTER_ARGS="-n" make clean-cluster'
            }
          }
        }

        stage("EKS") {
          agent {
            label "eks-integration-worker"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            DOCKER_IMAGE = "kubernetes-collector"
            AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
            AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
            INTEGRATION_TEST_ARGS = "all"
            INTEGRATION_TEST_BUILD = "ci"
          }
          steps {
            /* Setup */
            sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k eks'
            sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
            sh 'make target-eks'

            lock("integration-test-eks") {
              /* Collector Integration Tests */
              sh 'make clean-cluster'
              sh 'make -C collector integration-test'
            }

            lock("integration-test-eks") {
              /* Operator Integration Tests */
              sh 'make clean-cluster'
              /* k8s-events-only and common-metrics should not be run consecutively */
              sh 'make -C operator integration-test INTEGRATION_TEST_ARGS="-r k8s-events-only -r advanced -r common-metrics"'
              sh 'CLEAN_CLUSTER_ARGS="-n" make clean-cluster'
            }
          }
        }

        stage("AKS") {
          agent {
            label "aks-integration-worker"
          }
          options {
            timeout(time: 60, unit: 'MINUTES')
          }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            AKS_CLUSTER_NAME = "k8po-ci"
            DOCKER_IMAGE = "kubernetes-collector"
            INTEGRATION_TEST_ARGS="all"
            INTEGRATION_TEST_BUILD="ci"
          }
          steps {
            withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
              /* Setup */
              sh 'cd collector && ./hack/jenkins/setup-for-integration-test.sh -k aks'
              sh 'cd operator && ./hack/jenkins/setup-for-integration-test.sh'
              sh 'kubectl config use k8po-ci'

              lock("integration-test-aks") {
                /* Collector Integration Tests */
                sh 'make clean-cluster'
                sh 'make -C collector integration-test'
              }

              lock("integration-test-aks") {
                /* Operator Integration Tests */
                sh 'make clean-cluster'
                /* k8s-events-only and common-metrics should not be run consecutively */
                sh 'make -C operator integration-test INTEGRATION_TEST_ARGS="-r k8s-events-only -r validation-errors -r validation-legacy -r validation-errors-preprocessor-rules -r allow-legacy-install -r common-metrics"'
                sh 'CLEAN_CLUSTER_ARGS="-n" make clean-cluster'
              }
            }
          }
        }
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
