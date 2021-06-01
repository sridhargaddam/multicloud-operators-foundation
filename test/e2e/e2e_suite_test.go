package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/onsi/ginkgo/reporters"

	clustersetutils "github.com/open-cluster-management/multicloud-operators-foundation/pkg/utils/clusterset"
	"github.com/open-cluster-management/multicloud-operators-foundation/test/e2e/util"

	addonv1alpha1client "github.com/open-cluster-management/api/client/addon/clientset/versioned"
	clusterclient "github.com/open-cluster-management/api/client/cluster/clientset/versioned"
	hiveclient "github.com/openshift/hive/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	apiregistrationclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/typed/apiregistration/v1"
)

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	junit_report_file := os.Getenv("JUNIT_REPORT_FILE")
	if junit_report_file != "" {
		junitReporter := reporters.NewJUnitReporter(junit_report_file)
		ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "E2E suite", []ginkgo.Reporter{junitReporter})
	} else {
		ginkgo.RunSpecs(t, "E2E suite")
	}
}

const (
	eventuallyTimeout  = 300
	eventuallyInterval = 2
)

var (
	dynamicClient          dynamic.Interface
	kubeClient             kubernetes.Interface
	hiveClient             hiveclient.Interface
	clusterClient          clusterclient.Interface
	addonClient            addonv1alpha1client.Interface
	apiRegistrationClient  *apiregistrationclient.ApiregistrationV1Client
	managedClusterName     string
	managedClustersetName  string
	fakeManagedClusterName string
)

// This suite is sensitive to the following environment variables:
//
// - KUBECONFIG is the location of the kubeconfig file to use
// - MANAGED_CLUSTER_NAME is the name of managed cluster that is deployed by registration-operator
var _ = ginkgo.BeforeSuite(func() {
	var err error

	managedClusterName = os.Getenv("MANAGED_CLUSTER_NAME")
	if managedClusterName == "" {
		managedClusterName = "cluster1"
	}

	dynamicClient, err = util.NewDynamicClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	kubeClient, err = util.NewKubeClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	hiveClient, err = util.NewHiveClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	apiRegistrationClient, err = util.NewAPIServiceClient()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	cfg, err := util.NewKubeConfig()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	addonClient, err = addonv1alpha1client.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	clusterClient, err = clusterclient.NewForConfig(cfg)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	// accept the managed cluster that is deployed by registration-operator
	_, err = util.GetManagedCluster(dynamicClient, managedClusterName)
	if err != nil {
		err = util.AcceptManagedCluster(managedClusterName)
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	}

	// create a fake managed cluster
	fakeManagedCluster, err := util.CreateManagedCluster(dynamicClient)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	fakeManagedClusterName = fakeManagedCluster.GetName()

	gomega.Eventually(func() error {
		return util.CheckFoundationPodsReady()
	}, 60*time.Second, 2*time.Second).Should(gomega.Succeed())

	clusterset, err := clusterClient.ClusterV1alpha1().ManagedClusterSets().Create(context.Background(), util.ManagedClusterSetTemplate, metav1.CreateOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	//set ManagedClusterset for ManagedCluster
	clustersetlabel := map[string]string{
		clustersetutils.ClusterSetLabel: clusterset.GetName(),
	}
	managedClustersetName = clusterset.GetName()
	gomega.Eventually(func() error {
		managedCluster, err := util.GetClusterResource(dynamicClient, util.ManagedClusterGVR, managedClusterName)
		if err != nil {
			return err
		}
		err = util.AddLabels(managedCluster, clustersetlabel)
		if err != nil {
			return err
		}
		_, err = util.UpdateClusterResource(dynamicClient, util.ManagedClusterGVR, managedCluster)
		return err
	}, eventuallyTimeout, eventuallyInterval).Should(gomega.Succeed())

	//create clusterset admin clusterrolebinding
	_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(context.Background(), util.ClusterRoleBindingAdminTemplate, metav1.CreateOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	//create  clusterset view clusterrolebinding
	_, err = kubeClient.RbacV1().ClusterRoleBindings().Create(context.Background(), util.ClusterRoleBindingViewTemplate, metav1.CreateOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
})
