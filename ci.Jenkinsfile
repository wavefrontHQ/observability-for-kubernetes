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
      parallel {
        stage("Publish Collector") {
          agent { label 'worker-1' }
          options { timeout(time: 60, unit: 'MINUTES') }
          environment {
            RELEASE_TYPE = "alpha"
            DOCKER_IMAGE = "kubernetes-collector"
          }
          steps {
             sh './ci/jenkins/install_docker_buildx.sh'
             sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
             sh 'make -C collector clean docker-xplatform-build'
          }
        }

        stage("Publish Operator") {
          agent { label 'worker-2' }
          options { timeout(time: 60, unit: 'MINUTES') }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            RELEASE_TYPE = "alpha"
            COLLECTOR_PREFIX = "projects.registry.vmware.com/tanzu_observability_keights_saas"
            TOKEN = credentials('GITHUB_TOKEN')
            COLLECTOR_IMAGE = "kubernetes-collector"
          }
          steps {
            sh './ci/jenkins/setup-for-integration-test.sh -k gke'
            sh './ci/jenkins/install_docker_buildx.sh'
            sh 'echo $HARBOR_CREDS_PSW | docker login $PREFIX -u $HARBOR_CREDS_USR --password-stdin'
            sh 'make -C operator clean-build docker-xplatform-build'
            sh './operator/hack/jenkins/create-rc-ci.sh'
            script {
              env.OPERATOR_YAML_RC_SHA = sh(returnStdout: true, script: 'git rev-parse HEAD').trim()
            }
          }
        }

        stage("Collector Go Tests") {
          agent { label 'worker-3' }
          options { timeout(time: 60, unit: 'MINUTES') }
          steps {
            sh 'make -C collector checkfmt vet tests'
          }
        }

        stage("Operator Go Tests") {
          agent { label 'worker-4' }
          options { timeout(time: 60, unit: 'MINUTES') }
          steps {
            sh 'make -C operator checkfmt vet test golangci-lint'
          }
        }

        stage("Test Openshift build") {
          agent { label 'worker-5' }
          options { timeout(time: 60, unit: 'MINUTES') }
          steps {
            sh 'cd collector && docker build -f deploy/docker/Dockerfile-rhel .'
          }
        }
      }
    }

    stage('Run Collector Integration Tests') {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      environment {
        DOCKER_IMAGE = 'kubernetes-collector'
        INTEGRATION_TEST_ARGS = 'all'
        INTEGRATION_TEST_BUILD = 'ci'
      }
      // To save time, the integration tests and wavefront-metrics tests are split up between gke and eks
      // But we want to make sure that the combined and default integration tests are run on both
      parallel {
        stage("GKE") {
          agent { label 'worker-1' }
          options { timeout(time: 60, unit: 'MINUTES') }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
            GCP_ZONE = 'a'
          }
          steps {
            lock("integration-test-gke") {
              sh './ci/jenkins/setup-for-integration-test.sh -k gke'
              sh 'make gke-connect-to-cluster'
              sh 'make -C collector clean-cluster integration-test; make clean-cluster'
            }
          }
        }

        stage("EKS") {
          agent { label 'worker-2' }
          options { timeout(time: 60, unit: 'MINUTES') }
          environment {
            AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
            AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
          }
          steps {
            lock("integration-test-eks") {
              sh './ci/jenkins/setup-for-integration-test.sh -k eks'
              sh 'make target-eks'
              sh 'make -C collector clean-cluster integration-test; make clean-cluster'
            }
          }
        }

        stage("AKS") {
          agent { label 'worker-3' }
          options { timeout(time: 60, unit: 'MINUTES') }
          environment {
            AKS_CLUSTER_NAME = "k8po-ci"
          }
          steps {
            lock("integration-test-aks") {
              withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
                sh './ci/jenkins/setup-for-integration-test.sh'
                sh 'kubectl config use $AKS_CLUSTER_NAME'
                sh 'make -C collector clean-cluster integration-test; make clean-cluster'
              }
            }
          }
        }
      }
    }

    stage("Run Operator Integration Tests") {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      environment {
        OPERATOR_YAML_TYPE = 'rc'
        TOKEN = credentials('GITHUB_TOKEN')
      }

      parallel {
        stage("GKE") {
          agent { label 'worker-1' }
          options { timeout(time: 60, unit: 'MINUTES') }
          environment {
            GKE_CLUSTER_NAME = "k8po-jenkins-ci-zone-a"
            GCP_ZONE = 'a'
            GCP_CREDS = credentials("GCP_CREDS")
          }
          steps {
            sh './ci/jenkins/install_docker_buildx.sh'
            lock("integration-test-gke") {
              sh './ci/jenkins/setup-for-integration-test.sh -k gke'
              sh 'make gke-connect-to-cluster'
              sh 'make -C operator clean-cluster integration-test; make clean-cluster'
            }
          }
        }

        stage("EKS") {
          agent { label 'worker-2' }
          options { timeout(time: 60, unit: 'MINUTES') }
          environment {
            AWS_SHARED_CREDENTIALS_FILE = credentials("k8po-ci-aws-creds")
            AWS_CONFIG_FILE = credentials("k8po-ci-aws-profile")
            INTEGRATION_TEST_ARGS="-r advanced -r common-metrics"
          }
          steps {
            sh './ci/jenkins/install_docker_buildx.sh'
            lock("integration-test-eks") {
              sh './ci/jenkins/setup-for-integration-test.sh -k eks'
              sh 'make target-eks'
              sh 'make -C operator clean-cluster integration-test; make clean-cluster'
            }
          }
        }

        stage("AKS") {
          agent { label 'worker-4' }
          options { timeout(time: 60, unit: 'MINUTES') }
          environment {
            GCP_CREDS = credentials("GCP_CREDS")
            AKS_CLUSTER_NAME = "k8po-ci"
            INTEGRATION_TEST_ARGS = '-r validation-errors -r validation-legacy -r validation-errors-preprocessor-rules -r allow-legacy-install -r common-metrics'
          }
          steps {
            sh './ci/jenkins/install_docker_buildx.sh'
            lock("integration-test-aks") {
              withCredentials([file(credentialsId: 'aks-kube-config', variable: 'KUBECONFIG')]) {
                sh './ci/jenkins/setup-for-integration-test.sh'
                sh 'kubectl config use $AKS_CLUSTER_NAME'
                sh 'make -C operator clean-cluster integration-test; make clean-cluster'
              }
            }
          }
        }
      }
    }

    stage('Run TKGm Integration Tests') {
      when { beforeAgent true; expression { return env.RUN_CI == 'true' } }
      agent { label 'worker-5' }
      options { timeout(time: 60, unit: 'MINUTES') }
      environment {
        KUBECONFIG_DIR = "$HOME/.kube"
        KUBECONFIG_FILE = "$HOME/.kube/kubeconfig-tkgm"
        KUBECONFIG = "$KUBECONFIG_FILE:$HOME/.kube/config"
        TKGM_CONTEXT_NAME = 'tkg-mgmt-vc-admin@tkg-mgmt-vc'
      }

      stages {
        stage('Collector') {
          environment {
            DOCKER_IMAGE = 'kubernetes-collector'
            INTEGRATION_TEST_ARGS = 'all'
            INTEGRATION_TEST_BUILD = 'ci'
          }
          steps {
            lock("integration-test-tkgm") {
              sh './ci/jenkins/setup-for-integration-test.sh -k tkgm'
              sh 'kubectl config use-context $TKGM_CONTEXT_NAME ; kubectl get nodes'
              sh 'make -C collector clean-cluster integration-test; make clean-cluster'
            }
          }
        }

        stage('Operator') {
          environment {
            OPERATOR_YAML_TYPE = 'rc'
            TOKEN = credentials('GITHUB_TOKEN')
          }
          steps {
            lock("integration-test-tkgm") {
              sh './ci/jenkins/setup-for-integration-test.sh -k tkgm'
              sh 'kubectl config use-context $TKGM_CONTEXT_NAME ; kubectl get nodes'
              sh 'make -C operator clean-cluster integration-test; make clean-cluster'
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
