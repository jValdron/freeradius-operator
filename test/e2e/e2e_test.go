//go:build e2e
// +build e2e

/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/jValdron/freeradius-operator/test/utils"
)

// namespace where the project is deployed in
const namespace = "freeradius-operator-system"

// serviceAccountName created for the project
const serviceAccountName = "freeradius-operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "freeradius-operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "freeradius-operator-metrics-binding"

const radiusAuthCheckPod = "radius-auth-check"
const radiusE2ESecret = "e2e-test-secret"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "namespace", namespace, "--dry-run=client", "-o", "yaml")
		manifest, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to render manager namespace")
		cmd = exec.Command("kubectl", "apply", "-f", "-")
		cmd.Stdin = strings.NewReader(manifest)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to apply manager namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", managerImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("cleaning up the radius auth check pod")
		cmd = exec.Command("kubectl", "delete", "pod", radiusAuthCheckPod, "-n", "default", "--ignore-not-found")
		_, _ = utils.Run(cmd)

		By("removing e2e metrics ClusterRoleBinding")
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName, "--ignore-not-found")
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				By("getting the name of the controller-manager pod")
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				By("validating the pod's status")
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("ensuring a ClusterRoleBinding lets the manager service account scrape metrics")
			Expect(ensureMetricsReaderBinding()).To(Succeed())

			By("validating that the metrics service is available")
			cmd := exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("ensuring the controller pod is ready")
			verifyControllerPodReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pod", controllerPodName, "-n", namespace,
					"-o", "jsonpath={.status.conditions[?(@.type=='Ready')].status}")
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("True"), "Controller pod not ready")
			}
			Eventually(verifyControllerPodReady, 3*time.Minute, time.Second).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("Serving metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted, 3*time.Minute, time.Second).Should(Succeed())

			// +kubebuilder:scaffold:e2e-metrics-webhooks-readiness

			By("creating the curl-metrics pod to access the metrics endpoint")
			delCmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace, "--ignore-not-found", "--wait")
			_, _ = utils.Run(delCmd)

			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": [
								"for i in $(seq 1 30); do curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics && exit 0 || sleep 2; done; exit 1"
							],
							"securityContext": {
								"readOnlyRootFilesystem": true,
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccountName": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			verifyMetricsAvailable := func(g Gomega) {
				metricsOutput, err := getMetricsOutput()
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
				g.Expect(metricsOutput).NotTo(BeEmpty())
				g.Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
			}
			Eventually(verifyMetricsAvailable, 2*time.Minute).Should(Succeed())
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks

		It("should reconcile a Cluster into a FreeRADIUS Deployment", func() {
			const workloadNS = "default"
			const clusterName = "primary"

			By("installing cert-manager Issuer and Certificate for TLS")
			cmd := exec.Command("kubectl", "apply", "-k", "config/samples/cert-manager")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply cert-manager samples")

			waitCertReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "wait", "--for=condition=Ready",
					"certificate", "freeradius-primary-tls", "-n", workloadNS,
					"--timeout=10s")
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(waitCertReady, 3*time.Minute, 5*time.Second).Should(Succeed())

			By("applying sample Cluster")
			cmd = exec.Command("kubectl", "apply", "-f", "config/samples/freeradius_v1_cluster.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply Cluster sample")

			By("applying e2e RADIUS client and user")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/fixtures/radius_auth_client.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply e2e RADIUS client and user")

			deploymentName := fmt.Sprintf("freeradius-%s", clusterName)
			serviceHost := fmt.Sprintf("freeradius-%s.%s.svc.cluster.local", clusterName, workloadNS)

			By("waiting for generated FreeRADIUS Deployment")
			waitDeployment := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "deployment", deploymentName, "-n", workloadNS)
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(waitDeployment, 2*time.Minute, 3*time.Second).Should(Succeed())

			cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
				"deployment", deploymentName, "-n", workloadNS, "--timeout=5m")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "FreeRADIUS Deployment did not become Available")

			By("waiting for Cluster Ready status")
			waitClusterReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "wait", "--for=condition=Ready",
					"cluster", clusterName, "-n", workloadNS, "--timeout=15s")
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(waitClusterReady, 3*time.Minute, 5*time.Second).Should(Succeed())

			By("sending a PAP auth request with radclient")
			runRadiusAuthCheck(workloadNS, serviceHost)
		})

		It("should authenticate inner-tunnel users against LDAP", func() {
			const workloadNS = "default"
			const clusterName = "ldap"

			By("deploying OpenLDAP and LDAP-enabled Cluster")
			// Replace prior revisions that mounted bootstrap LDIF into slapd (osixia exits with LDAP 68).
			for _, args := range [][]string{
				{"delete", "job", "openldap-seed", "-n", workloadNS, "--ignore-not-found", "--wait"},
				{"delete", "deployment", "openldap", "-n", workloadNS, "--ignore-not-found", "--wait"},
			} {
				del := exec.Command("kubectl", append([]string{}, args...)...)
				_, _ = utils.Run(del)
			}

			cmd := exec.Command("kubectl", "apply", "-k", "test/e2e/fixtures/ldap")
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to apply LDAP fixtures")

			waitOpenLDAP := func(g Gomega) {
				cmd := exec.Command("kubectl", "wait", "--for=condition=Available",
					"deployment", "openldap", "-n", workloadNS, "--timeout=15s")
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(waitOpenLDAP, 5*time.Minute, 5*time.Second).Should(Succeed())

			By("waiting for OpenLDAP seed job")
			waitSeedJob := func(g Gomega) {
				cmd := exec.Command("kubectl", "wait", "--for=condition=complete",
					"job", "openldap-seed", "-n", workloadNS, "--timeout=30s")
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(waitSeedJob, 3*time.Minute, 5*time.Second).Should(Succeed())

			waitCertReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "wait", "--for=condition=Ready",
					"certificate", "freeradius-ldap-tls", "-n", workloadNS, "--timeout=10s")
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(waitCertReady, 3*time.Minute, 5*time.Second).Should(Succeed())

			By("applying e2e RADIUS client only (no local User CR)")
			cmd = exec.Command("kubectl", "apply", "-f", "test/e2e/fixtures/radius_auth_client.yaml")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			deploymentName := fmt.Sprintf("freeradius-%s", clusterName)

			By("waiting for LDAP FreeRADIUS Deployment")
			waitClusterReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "wait", "--for=condition=Ready",
					"cluster", clusterName, "-n", workloadNS, "--timeout=15s")
				_, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
			}
			Eventually(waitClusterReady, 5*time.Minute, 5*time.Second).Should(Succeed())

			cmd = exec.Command("kubectl", "wait", "--for=condition=Available",
				"deployment", deploymentName, "-n", workloadNS, "--timeout=5m")
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred())

			By("checking LDAP user auth on inner-tunnel via radclient in the FreeRADIUS pod")
			runInnerTunnelLDAPAuth(workloadNS, clusterName, "e2eldap", "e2eldappass")
		})
	})
})

// runRadiusAuthCheck runs radclient in a one-shot pod and expects Access-Accept.
func runRadiusAuthCheck(workloadNS, serviceHost string) {
	script := fmt.Sprintf(`for i in $(seq 1 30); do
	if printf 'User-Name = test, User-Password = testpass\n' | /opt/bin/radclient -x %s:1812 auth %s 2>&1 | tee /dev/stderr | grep -q 'Access-Accept'; then
		exit 0
	fi
	sleep 2
done
exit 1`, serviceHost, radiusE2ESecret)

	overrides := fmt.Sprintf(`{
		"spec": {
			"containers": [{
				"name": "radclient",
				"image": "freeradius/freeradius-server:3.2.10-alpine",
				"command": ["/bin/sh", "-c"],
				"args": [%q],
				"securityContext": {
					"allowPrivilegeEscalation": false,
					"capabilities": { "drop": ["ALL"] },
					"seccompProfile": { "type": "RuntimeDefault" }
				}
			}],
			"restartPolicy": "Never"
		}
	}`, script)

	deleteCmd := exec.Command("kubectl", "delete", "pod", radiusAuthCheckPod, "-n", workloadNS, "--ignore-not-found", "--wait")
	_, _ = utils.Run(deleteCmd)

	cmd := exec.Command("kubectl", "run", radiusAuthCheckPod,
		"--restart=Never",
		"--namespace", workloadNS,
		"--image=freeradius/freeradius-server:3.2.10-alpine",
		"--overrides", overrides,
	)
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to create radius-auth-check pod")

	verify := func(g Gomega) {
		cmd := exec.Command("kubectl", "get", "pod", radiusAuthCheckPod,
			"-n", workloadNS, "-o", "jsonpath={.status.phase}")
		phase, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(phase).To(Equal("Succeeded"), "radclient pod did not succeed")

		cmd = exec.Command("kubectl", "logs", radiusAuthCheckPod, "-n", workloadNS)
		logs, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(logs).To(ContainSubstring("Access-Accept"), "expected RADIUS Access-Accept")
	}
	Eventually(verify, 3*time.Minute, 3*time.Second).Should(Succeed())
}

// runInnerTunnelLDAPAuth execs radclient in the FreeRADIUS pod (inner-tunnel listens on 127.0.0.1:18120).
func runInnerTunnelLDAPAuth(workloadNS, clusterName, username, password string) {
	podName, err := freeradiusPodName(workloadNS, clusterName)
	Expect(err).NotTo(HaveOccurred())

	script := fmt.Sprintf(`for i in $(seq 1 30); do
	if printf 'User-Name = %s, User-Password = %s\n' | /opt/bin/radclient -x 127.0.0.1:18120 auth testing123 2>&1 | tee /dev/stderr | grep -q 'Access-Accept'; then
		exit 0
	fi
	sleep 2
done
exit 1`, username, password)

	verify := func(g Gomega) {
		cmd := exec.Command("kubectl", "exec", "-n", workloadNS, podName, "-c", "freeradius",
			"--", "/bin/sh", "-c", script)
		out, err := utils.Run(cmd)
		g.Expect(err).NotTo(HaveOccurred(), "radclient exec failed: %s", out)
	}
	Eventually(verify, 3*time.Minute, 5*time.Second).Should(Succeed())
}

func freeradiusPodName(ns, clusterName string) (string, error) {
	cmd := exec.Command("kubectl", "get", "pods", "-n", ns,
		"-l", fmt.Sprintf("app=freeradius,freeradius.valdron.ca/cluster=%s", clusterName),
		"-o", "go-template={{ range .items }}{{ if not .metadata.deletionTimestamp }}{{ .metadata.name }}{{ \"\\n\" }}{{ end }}{{ end }}")
	out, err := utils.Run(cmd)
	if err != nil {
		return "", err
	}
	names := utils.GetNonEmptyLines(out)
	if len(names) != 1 {
		return "", fmt.Errorf("expected 1 freeradius pod, got %d", len(names))
	}
	return names[0], nil
}

func ensureMetricsReaderBinding() error {
	manifest := fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: freeradius-operator-metrics-reader
subjects:
- kind: ServiceAccount
  name: %s
  namespace: %s
`, metricsRoleBindingName, serviceAccountName, namespace)
	path := filepath.Join(os.TempDir(), metricsRoleBindingName+".yaml")
	if err := os.WriteFile(path, []byte(manifest), 0o600); err != nil {
		return err
	}
	cmd := exec.Command("kubectl", "apply", "-f", path)
	_, err := utils.Run(cmd)
	return err
}

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	By("creating temporary file to store the token request")
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		By("executing kubectl command to create the token")
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		By("parsing the JSON output to extract the token")
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() (string, error) {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	return utils.Run(cmd)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
