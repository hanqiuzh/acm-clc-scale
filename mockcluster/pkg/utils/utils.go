package utils

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"text/template"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

type Recipient struct {
	Name, Gift string
	Attended   bool
}

func getNamespace(clusterName string) (*unstructured.Unstructured, error) {
	yamlData := `
apiVersion: v1
kind: Namespace
metadata:
  name: {{.ClusterName}}
  labels:
    cluster.open-cluster-management.io/managedCluster: {{.ClusterName}}
`
	type Value struct {
		ClusterName string
	}
	r := Value{ClusterName: clusterName}
	return yamlToObj(yamlData, r)
}

func getManagedCluster(clusterName string) (*unstructured.Unstructured, error) {
	/* 	yamlData := `
	apiVersion: cluster.open-cluster-management.io/v1
	kind: ManagedCluster
	metadata:
	  labels:
	    cloud: auto-detect
	    vendor: auto-detect
	    name: {{.ClusterName}}
	    mock-cluster: "true"
	  name: local-cluster
	spec:
	  hubAcceptsClient: true
	`
		type Value struct {
			ClusterName string
		}
		r := Value{ClusterName: clusterName} */

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cluster.open-cluster-management.io/v1",
			"kind":       "ManagedCluster",
			"metadata": map[string]interface{}{
				"name": clusterName,
				"labels": map[string]interface{}{
					"cloud":        "auto-detect",
					"vendor":       "auto-detect",
					"name":         clusterName,
					"mock-cluster": "true",
				},
			},
			"spec": map[string]interface{}{
				"hubAcceptsClient": true,
			},
			"status": map[string]interface{}{
				"conditions": []map[string]interface{}{
					map[string]interface{}{
						"type":               "placeholder",
						"lastTransitionTime": "2020-01-01T01:01:01Z",
						"reason":             "placeholder",
						"status":             "False",
					},
				},
			},
		},
	}, nil
}

func yamlToObj(yamlData string, r interface{}) (*unstructured.Unstructured, error) {
	t := template.Must(template.New("manifest").Parse(yamlData))
	buf := bytes.NewBufferString("")
	err := t.Execute(buf, r)
	if err != nil {
		log.Println("executing template:", err)
		return nil, err
	}
	obj := &unstructured.Unstructured{}

	// decode YAML into unstructured.Unstructured
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	_, _, err = dec.Decode([]byte(buf.String()), nil, obj)
	return obj, err
}
func getKlusterletAddonConfig(clusterName string) (*unstructured.Unstructured, error) {
	yamlData := `
apiVersion: agent.open-cluster-management.io/v1
kind: KlusterletAddonConfig
metadata:
  name: {{.ClusterName}}
  namespace: {{.ClusterName}}
  labels:
  mock-cluster: "true"
spec:
  clusterLabels:
    cloud: auto-detect
    vendor: auto-detect
  clusterName: {{.ClusterName}}
  clusterNamespace: {{.ClusterName}}
  applicationManager:
    enabled: true
  certPolicyController:
    enabled: true
  iamPolicyController:
    enabled: true
  policyController:
    enabled: true
  searchCollector:
    enabled: true
  version: 2.0.0
`
	type Value struct {
		ClusterName string
	}
	r := Value{ClusterName: clusterName}
	return yamlToObj(yamlData, r)
}

type MockClusterClient struct {
	Client              dynamic.Interface
	Prefix              string
	SpokeClientRegistry map[string]dynamic.Interface
	mux                 *sync.Mutex
}

func (m *MockClusterClient) GetClient(name string) dynamic.Interface {
	if name == "" {
		return m.Client
	}

	m.mux.Lock()
	defer m.mux.Unlock()

	sClt, ok := m.SpokeClientRegistry[name]
	if !ok {
		return m.Client
	}

	return sClt
}

func NewMockClusterClient(hubClt dynamic.Interface, prefix string) *MockClusterClient {
	return &MockClusterClient{
		Client:              hubClt,
		Prefix:              prefix,
		SpokeClientRegistry: map[string]dynamic.Interface{},
		mux:                 &sync.Mutex{},
	}
}

func (m *MockClusterClient) UpdateSpokeClientRegistry(sClt dynamic.Interface, name string) {
	m.mux.Lock()
	defer m.mux.Unlock()

	m.SpokeClientRegistry[name] = sClt
}

var gvrKlusterletAddonConfig = schema.GroupVersionResource{Group: "agent.open-cluster-management.io", Version: "v1", Resource: "klusterletaddonconfigs"}
var gvrManifestwork = schema.GroupVersionResource{Group: "work.open-cluster-management.io", Version: "v1", Resource: "manifestworks"}
var gvrManagedCluster = schema.GroupVersionResource{Group: "cluster.open-cluster-management.io", Version: "v1", Resource: "managedclusters"}

//var gvrClusterManagementAddOn = schema.GroupVersionResource{Group: "addon.open-cluster-management.io", Version: "v1alpha1", Resource: "clustermanagementaddons"}
//var gvrManagedClusterAddOn = schema.GroupVersionResource{Group: "addon.open-cluster-management.io", Version: "v1alpha1", Resource: "managedclusteraddons"}
var gvrLease = schema.GroupVersionResource{Group: "coordination.k8s.io", Version: "v1", Resource: "leases"}
var gvrNamespace = schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}

func (m *MockClusterClient) CreateMockManagedCluster(name string) (err error) {
	// createt namespace
	fmt.Println("Creating ns")
	nsObj, err := getNamespace(name)
	if err != nil {
		fmt.Printf("error %v\n", err)
		return err
	}
	_, err = m.Client.Resource(gvrNamespace).Namespace("").Create(context.TODO(), nsObj, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("error %v\n", err)
		return err
	}
	fmt.Println("Creating managedcluster")
	// create managedcluster
	mc, err := getManagedCluster(name)
	if err != nil {
		fmt.Printf("error get managedcluster %v\n", err)
		return err
	}
	_, err = m.Client.Resource(gvrManagedCluster).Namespace("").Create(context.TODO(), mc, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("error creating managedcluster: %v\n", err)
		return err
	}
	fmt.Println("Creating klusterletaddonconfig")
	// create klusterletaddonconfig
	k, err := getKlusterletAddonConfig(name)
	if err != nil {
		fmt.Printf("error %v\n", err)
		return err
	}
	_, err = m.Client.Resource(gvrKlusterletAddonConfig).Namespace(name).Create(context.TODO(), k, metav1.CreateOptions{})
	if err != nil {
		fmt.Printf("error %v\n", err)
		return err
	}
	fmt.Println("Creating lease")
	leaseNames := []string{
		"application-manager",
		"cert-policy-controller",
		"iam-policy-controller",
		"policy-controller",
		"search-collector",
		"work-manager",
	}
	for _, leaseName := range leaseNames {
		currentTime := time.Now().Format("2006-01-02T15:04:05.000000Z07:00")
		lease := newLease(leaseName, name, currentTime)
		if _, err := m.Client.Resource(gvrLease).Namespace(name).Create(context.TODO(), lease, metav1.CreateOptions{}); err != nil {
			fmt.Printf("error %s\n", err)
			return err
		}
	}

	return nil
}

func (m *MockClusterClient) DetachMockManagedCluster(name string) (err error) {
	return m.Client.Resource(gvrManagedCluster).Namespace("").Delete(context.TODO(), name, metav1.DeleteOptions{})
}

func (m *MockClusterClient) ListMockClusters() ([]string, error) {
	res := make([]string, 0)
	items, err := m.Client.Resource(gvrManagedCluster).Namespace("").List(context.TODO(), metav1.ListOptions{LabelSelector: "mock-cluster=true"})
	if err != nil {
		fmt.Printf("%v\n", err)
		return nil, err
	}
	j := 0
	for _, v := range items.Items {
		// filter with prefix
		name := v.GetName()
		if strings.HasPrefix(name, m.Prefix) {
			res = append(res, name)
			j++
		}
	}

	return res, nil
}

func generateManifestStatus(ordinal int, applied string) string {
	var stamp int = 1
	fixTime := time.Date(stamp+2020, time.January, stamp, stamp, stamp, stamp, stamp, &time.Location{}).Format("2006-01-02T15:04:05.000000Z07:00")
	return fmt.Sprintf(`{"conditions":[{"lastTransitionTime":"%s","reason":"mock","message":"mock","type":"Applied","status":"%s"}],"resourceMeta":{"ordinal":%d}}`, fixTime, applied, ordinal)
}

func generateManifestTopLevelStatus(applied string) string {
	var stamp int = 1
	fixTime := time.Date(stamp+2020, time.January, stamp, stamp, stamp, stamp, stamp, &time.Location{}).Format("2006-01-02T15:04:05.000000Z07:00")
	return fmt.Sprintf(`"conditions":[{"lastTransitionTime":"%s","reason":"mock","message":"mock","type":"Applied","status":"%s"},{"lastTransitionTime":"%s","reason":"mock","message":"mock","type":"Available","status":"%s"}]`, fixTime, applied, fixTime, applied)
}

func setManifestWorkAppliedStatus(clientHubDynamic dynamic.Interface, name, namespace string, succeed int, failed int) error {
	ordinal := 0
	patchString := `{"status":{"resourceStatus":{"manifests":[`

	for i := 0; i < failed; i++ {
		ordinal = i
		s := generateManifestStatus(ordinal, "False")
		if i < failed-1 || succeed > 0 {
			s = s + ","
		}
		patchString = patchString + s
	}
	for i := 0; i < succeed; i++ {
		ordinal = i + failed
		s := generateManifestStatus(ordinal, "True")
		if i < succeed-1 {
			s = s + ","
		}
		patchString = patchString + s
	}

	patchString = patchString + `]},` + generateManifestTopLevelStatus("True") + `}}`

	_, err := clientHubDynamic.Resource(gvrManifestwork).Namespace(namespace).Patch(context.TODO(), name, types.MergePatchType, []byte(patchString), metav1.PatchOptions{}, "status")
	return err
}

func (m *MockClusterClient) MarkManifestWorksInstalled(name string) error {
	l, err := m.Client.Resource(gvrManifestwork).Namespace(name).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("error %v\n", err)
		return err
	}
	var retErr error = nil
	for _, item := range l.Items {
		if err := setManifestWorkAppliedStatus(m.GetClient(name), item.GetName(), name, 10, 0); err != nil {
			retErr = err
		}
	}
	return retErr
}
func (m *MockClusterClient) SetClusterOnline(name string) error {
	patchString := `{"status":` +
		`{"conditions":[` +
		`{"type":"ManagedClusterJoined","lastTransitionTime":"2020-01-01T01:01:01Z","message":"Managed Cluster joined","status":"True","reason":"ManagedClusterJoined"}` + `,` +
		`{"type":"ManagedClusterConditionAvailable","lastTransitionTime":"2020-01-01T01:01:01Z","message":"Managed Cluster Available","status":"True","reason":"ManagedClusterConditionAvailable"}` + `,` +
		`{"type":"HubAcceptedManaged","lastTransitionTime":"2020-01-01T01:01:01Z","message":"Accepted by hub cluster admin","status":"True","reason":"HubClusterAdminAccepted"}` + `,` +
		`{"type":"HubAcceptedManagedCluster","lastTransitionTime":"2020-01-01T01:01:01Z","message":"Accepted by hub cluster admin","status":"True","reason":"HubClusterAdminAccepted"}` +
		`]}}`

	_, err := m.GetClient(name).Resource(gvrManagedCluster).Namespace("").Patch(context.TODO(), name, types.MergePatchType, []byte(patchString), metav1.PatchOptions{}, "status")
	return err

}

func newLease(name, namespace string, renewTime string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "coordination.k8s.io/v1",
			"kind":       "Lease",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"leaseDurationSeconds": 60,
				"renewTime":            renewTime,
			},
		},
	}
}

func (m *MockClusterClient) UpdateLeases(name string) error {
 	// list leases in that namespace
	l, err := m.Client.Resource(gvrLease).Namespace(name).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		fmt.Printf("error %v\n", err)
		return err
	}
	currentTime := time.Now().Format("2006-01-02T15:04:05.000000Z07:00")
	patchString := `{"spec":{"renewTime":"` + currentTime + `"}}`
	// update leases to current timestamp
	var retErr error = nil
	for _, item := range l.Items {
		n := item.GetName()
		if _, err :=  m.GetClient(name).Resource(gvrLease).Namespace(name).Patch(context.TODO(), n, types.MergePatchType, []byte(patchString), metav1.PatchOptions{}); err != nil {
			retErr = err
		}
	}

	return retErr
}

var uwg sync.WaitGroup

const keepAliveConcurrentNum = 50
const maxQueue = 5000

type KeepAliveWorkQueue struct {
	IsInQueue map[string]bool
	workQueue chan string
	mu        sync.Mutex
}

func createKeepAliveWorkQueue() *KeepAliveWorkQueue {
	return &KeepAliveWorkQueue{
		IsInQueue: make(map[string]bool),
		workQueue: make(chan string, maxQueue),
	}
}
func (q *KeepAliveWorkQueue) GetWork() string {
	res := <-q.workQueue
	q.mu.Lock()
	q.IsInQueue[res] = false
	q.mu.Unlock()
	return res
}
func (q *KeepAliveWorkQueue) PutWork(name string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.IsInQueue[name] {
		return false
	}
	q.IsInQueue[name] = true
	q.workQueue <- name //non blocked
	return true
}

func (m *MockClusterClient) startKeepAliveWorker(q *KeepAliveWorkQueue) {
	for {
		cluster := q.GetWork()
		fmt.Println("updating " + cluster)
		// check if there is any manifestwork we need to set installed
		if err := m.MarkManifestWorksInstalled(cluster); err != nil {
			fmt.Printf("error %s\n", err)
		}
		// set managedcluster to online
		if err := m.SetClusterOnline(cluster); err != nil {
			fmt.Printf("error %s\n", err)
		}
		// update lease of managedcluster
		if err := m.UpdateLeases(cluster); err != nil {
			fmt.Printf("error %s\n", err)
		}
	}
}

// keep alive all will make sure all managedcluster's lease & addons' lease updated every minute
func (m *MockClusterClient) KeepAliveAll(numConcurrent int) {
	clusterNames := []string{}
	// create work queue
	q := createKeepAliveWorkQueue()

	// init go subroutines
	if numConcurrent < 1 {
		numConcurrent = 1
	}
	for i := 0; i < numConcurrent; i++ {
		fmt.Println("creating worker")
		go m.startKeepAliveWorker(q)
	}

	for {
		// get all mocked managedclusters
		newClusterNames, err := m.ListMockClusters()
		if err != nil {
			fmt.Printf("error %s\n", err)
		} else {
			clusterNames = newClusterNames
		}
		fmt.Printf("will update %d clusters\n", len(clusterNames))
		rand.Shuffle(len(clusterNames), func(i, j int) {
			clusterNames[i], clusterNames[j] = clusterNames[j], clusterNames[i]
		})
		// push to queue
		numAddedWork := 0
		for _, cluster := range clusterNames {
			if q.PutWork(cluster) {
				numAddedWork++
			}
		}
		fmt.Printf("added %d work into workqueue, %d are already in queue\n", numAddedWork, len(clusterNames)-numAddedWork)

		time.Sleep(60 * time.Second)
	}

}
