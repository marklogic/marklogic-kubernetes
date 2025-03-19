/* groovylint-disable CompileStatic, LineLength, VariableTypeRequired */
// This Jenkinsfile defines internal MarkLogic build pipeline.

//Shared library definitions: https://github.com/marklogic/MarkLogic-Build-Libs/tree/1.0-declarative/vars
@Library('shared-libraries@1.0-declarative')
import groovy.json.JsonSlurperClassic

emailList = 'vitaly.korolev@progress.com, sumanth.ravipati@progress.com, peng.zhou@progress.com, fayez.saliba@progress.com, barkha.choithani@progress.com, romain.winieski@progress.com'
emailSecList = 'Rangan.Doreswamy@progress.com, Mahalakshmi.Srinivasan@progress.com'
gitCredID = 'marklogic-builder-github'
JIRA_ID = ''
JIRA_ID_PATTERN = /(?i)(MLE)-\d{3,6}/
LINT_OUTPUT = ''
SCAN_OUTPUT = ''
IMAGE_INFO = 0

// Define local funtions
void preBuildCheck() {
    // Initialize parameters as env variables as workaround for https://issues.jenkins-ci.org/browse/JENKINS-41929
    evaluate """${ def script = ''; params.each { k, v -> script += "env.${k } = '''${v}'''\n" }; return script}"""

    JIRA_ID = extractJiraID()
    echo 'Jira ticket number: ' + JIRA_ID

    if (env.GIT_URL) {
        githubAPIUrl = GIT_URL.replace('.git', '').replace('github.com', 'api.github.com/repos')
        echo 'githubAPIUrl: ' + githubAPIUrl
    } else {
        echo 'Warning: GIT_URL is not defined'
    }

    if (env.CHANGE_ID) {
        if (prDraftCheck()) { sh 'exit 1' }
        if (getReviewState().equalsIgnoreCase('CHANGES_REQUESTED')) {
            echo 'PR changes requested. (' + reviewState + ') Aborting.'
            sh 'exit 1'
        }
    }

    // our VMs sometime disable bridge traffic. this should help to restore it.
    sh 'sudo sh -c "echo 1 > /proc/sys/net/bridge/bridge-nf-call-iptables"'

    // install local version of golangci-lint and gotestsum
    sh '''
        curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b /space/go/bin v1.50.0
        wget https://github.com/gotestyourself/gotestsum/releases/download/v1.12.0/gotestsum_1.12.0_linux_amd64.tar.gz -O gotestsum.tar.gz
        mkdir -p /space/go/bin/
        tar -xf gotestsum.tar.gz -C /space/go/bin/ gotestsum
    '''
}

@NonCPS
def extractJiraID() {
    // Extract Jira ID from one of the environment variables
    def match
    if (env.CHANGE_TITLE) {
        match = env.CHANGE_TITLE =~ JIRA_ID_PATTERN
    }
    else if (env.BRANCH_NAME) {
        match = env.BRANCH_NAME =~ JIRA_ID_PATTERN
    }
    else if (env.GIT_BRANCH) {
        match = env.GIT_BRANCH =~ JIRA_ID_PATTERN
    }
    else {
        echo 'Warning: No Git title or branch available.'
        return ''
    }
    try {
        return match[0][0]
    } catch (any) {
        echo 'Warning: Jira ticket number not detected.'
        return ''
    }
}

def prDraftCheck() {
    withCredentials([usernameColonPassword(credentialsId: gitCredID, variable: 'Credentials')]) {
        PrObj = sh(returnStdout: true, script:'''
                    curl -s -u $Credentials  -X GET  ''' + githubAPIUrl + '''/pulls/$CHANGE_ID
                    ''')
    }
    def jsonObj = new JsonSlurperClassic().parseText(PrObj.toString().trim())
    return jsonObj.draft
}

def getReviewState() {
    def reviewResponse
    def commitHash
    withCredentials([usernameColonPassword(credentialsId: gitCredID, variable: 'Credentials')]) {
        reviewResponse = sh(returnStdout: true, script:'''
                            curl -s -u $Credentials  -X GET  ''' + githubAPIUrl + '''/pulls/$CHANGE_ID/reviews
                            ''')
        commitHash = sh(returnStdout: true, script:'''
                        curl -s -u $Credentials  -X GET  ''' + githubAPIUrl + '''/pulls/$CHANGE_ID
                        ''')
    }
    def jsonObj = new JsonSlurperClassic().parseText(commitHash.toString().trim())
    def commitId = jsonObj.head.sha
    println(commitId)
    def reviewState = getReviewStateOfPR reviewResponse, 2, commitId
    echo reviewState
    return reviewState
}

void resultNotification(message) {
    def author, authorEmail, emailList
    if (env.CHANGE_AUTHOR) {
        author = env.CHANGE_AUTHOR.toString().trim().toLowerCase()
        authorEmail = getEmailFromGITUser author
        emailList = params.emailList + ',' + authorEmail
    } else {
        emailList = params.emailList
    }
    jira_link = "https://progresssoftware.atlassian.net/browse/${JIRA_ID}"
    email_body = "<b>Jenkins pipeline for</b> ${env.JOB_NAME} <br><b>Build Number: </b>${env.BUILD_NUMBER} <br><br><b>Lint Output: </b><br><pre><code>${LINT_OUTPUT}</code></pre><br><br><b>Scan Output: </b><br><pre><code>${SCAN_OUTPUT}</code></pre><br><br><b>Build URL: </b><br><a href='${env.BUILD_URL}'>${env.BUILD_URL}</a>"
    jira_email_body = "${email_body} <br><br><b>Jira URL: </b><br><a href='${jira_link}'>${jira_link}</a>"

    if (JIRA_ID) {
        def comment = [ body: "Jenkins pipeline build result: ${message}" ]
        jiraAddComment site: 'JIRA', idOrKey: JIRA_ID, failOnError: false, input: comment
        mail charset: 'UTF-8', mimeType: 'text/html', to: "${emailList}", body: "${jira_email_body}", subject: "${message}: ${env.JOB_NAME} #${env.BUILD_NUMBER} - ${JIRA_ID}"
    } else {
        mail charset: 'UTF-8', mimeType: 'text/html', to: "${emailList}", body: "${email_body}", subject: "${message}: ${env.JOB_NAME} #${env.BUILD_NUMBER}"
    }
}

void lint() {
    sh '''
        make lint saveOutput=true
    '''

    LINT_OUTPUT = sh(returnStdout: true, script: 'echo helm template lint output: ;cat helm-lint-output.txt ;echo all tests lint output: ;cat test-lint-output.txt').trim()

    sh '''
        rm -f helm-lint-output.txt test-lint-output.txt
    '''
}

void imageScan() {
    sh "make image-scan saveOutput=true"

    SCAN_OUTPUT = sh(returnStdout: true, script:'cat dep-image-scan.txt')
    hasCriticalOrHigh = SCAN_OUTPUT.contains("High") || SCAN_OUTPUT.contains("Critical")
    if (hasCriticalOrHigh) {
        mail charset: 'UTF-8', mimeType: 'text/html', to: "${emailSecList}", body: "<br>Jenkins pipeline for ${env.JOB_NAME} <br>Build Number: ${env.BUILD_NUMBER} <br>Vulnerabilities: <pre><code>${SCAN_OUTPUT}</code></pre>", subject: "Critical or High Security Vulnerabilities Found: ${env.JOB_NAME} #${env.BUILD_NUMBER}"
    }

    sh '''rm -f dep-image-scan.txt'''
}

void publishTestResults() {
    junit allowEmptyResults:true, testResults: '**/test/test_results/*.xml'
    archiveArtifacts artifacts: '**/test/test_results/*.xml', allowEmptyArchive: true
}

pipeline {
    agent {
        label {
            label 'cld-kubernetes'
        }
    }
    options {
        checkoutToSubdirectory '.'
        buildDiscarder logRotator(artifactDaysToKeepStr: '7', artifactNumToKeepStr: '', daysToKeepStr: '30', numToKeepStr: '')
        skipStagesAfterUnstable()
    }
    triggers {
        parameterizedCron( env.BRANCH_NAME == 'develop' ? '''00 04 * * * % IMAGE_SCAN=true;HELM_UPGRADE_TESTS=true;HC_TESTS=true
                                                             00 04 * * * % dockerImageType=ubi''' : '')
    }
    environment {
        dockerRegistry = 'ml-docker-db-dev-tierpoint.bed-artifactory.bedford.progress.com'
        dockerRepository = "${dockerRegistry}/marklogic/marklogic-server-${params.dockerImageType}"
        PATH = "/space/go/bin:${env.PATH}"
        MINIKUBE_HOME = "/space/minikube/"
        KUBECONFIG = "/space/.kube-config"
        GOPATH = "/space/go"
    }

    parameters {
        choice(name: 'dockerImageType', choices: 'ubi-rootless\nubi\nubi9-rootless\nubi9', description: 'Platform type for Docker image')
        string(name: 'dockerVersion', defaultValue: 'latest-11', description: 'Docker tag to use for tests. (e.g. 11.2.nightly-ubi-rootless-1.1.2) Has to correspond with dockerImageType.', trim: true)
        string(name: 'prevDockerVersion', defaultValue: 'latest-10', description: 'Previous Docker version for MarkLogic upgrade tests. (e.g. 10.0-10.2-centos-1.1.2) Has to correspond with dockerImageType.', trim: true)
        choice(name: 'K8_VERSION', choices: 'v1.31.7\nv1.32.3\nv1.30.11\nv1.29.15\nv1.28.15\nv1.27.16\nv1.26.15\nv1.25.16', description: 'Test Kubernetes version.')
        booleanParam(name: 'KUBERNETES_TESTS', defaultValue: true, description: 'Run kubernetes tests')
        string(name: 'KUBERNETES_TEST_SELECTION', defaultValue: '...', description: 'Pick one test to run. (e.g. tls_test.go) ... will run all tests.', trim: true)
        booleanParam(name: 'HC_TESTS', defaultValue: false, description: 'Run Hub Central E2E UI tests (takes about 3 hours)')
        booleanParam(name: 'IMAGE_SCAN', defaultValue: false, description: 'Find and scan dependent Docker images for security vulnerabilities')
        booleanParam(name: 'HELM_UPGRADE_TESTS', defaultValue: false, description: 'Run Helm upgrade in E2E tests (runs nightly on develop)')
        string(name: 'InitialChartVersion', defaultValue: '1.1.2', description: 'Helm Chart Version to use for upgrade tests. (e.g. 1.1.2)', trim: true)
        string(name: 'emailList', defaultValue: emailList, description: 'List of email for build notification', trim: true)
    }

    stages {
        stage('Pre-Build-Check') {
            steps {
                preBuildCheck()
            }
        }

        stage('Lint') {
            steps {
                lint()
            }
        }

        stage('Image-Scan') {
            when {
                expression { return params.IMAGE_SCAN }
            }
            steps {
                imageScan()
            }
        }

        stage('Kubernetes-Run-Tests') {
            when {
                expression { return params.KUBERNETES_TESTS }
            }
            steps {
                sh """
                    make test dockerImage=${dockerRepository}:${dockerVersion} prevDockerImage=${dockerRepository}:${prevDockerVersion} kubernetesVersion=${params.K8_VERSION} saveOutput=true minikubeMemory=20gb testSelection=${params.KUBERNETES_TEST_SELECTION}
                """
            }
        }
        stage('Kubernetes-Run-Upgrade-Tests') {
            when {
                expression { return params.HELM_UPGRADE_TESTS }
            }
            steps {
                sh """
                    export upgradeTest=true; export initialChartVersion=${params.InitialChartVersion}; make upgrade-test dockerImage=${dockerRepository}:${dockerVersion} prevDockerImage=${dockerRepository}:${prevDockerVersion} kubernetesVersion=${params.K8_VERSION} saveOutput=true minikubeMemory=20gb
                """
            }
        }
        stage('Kubernetes-Run-HC-Tests') {
            when {
                expression { return params.HC_TESTS }
            }
            steps {
                sh """
                    make hc-test dockerImage=${dockerRepository}:${dockerVersion} kubernetesVersion=${params.K8_VERSION} minikubeMemory=20gb
                """
            }
        }
    }

    post {
        always {
            publishTestResults()
            sh '''
	            sudo sysctl -w vm.nr_hugepages=0
                minikube delete --all --purge
                docker stop $(docker ps -a -q) || true
                docker system prune --force --all
                docker volume prune --force --all
                docker system df
                sudo rm -rf /space/minikube/ /space/go /space/.kube-config
            '''
            sh "rm -rf $WORKSPACE/test/test_results/"
        }
        success {
            resultNotification('BUILD SUCCESS ✅')
        }
        failure {
            resultNotification('BUILD ERROR ❌')
        }
        unstable {
            resultNotification('BUILD UNSTABLE ❌')
        }
    }
}