package main

import (
	"flag"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"github.com/prometheus/common/version"
	"sync"
)

var (
	listeningAddress = flag.String("telemetry.address", ":9309", "Address on which to expose metrics.")
	metricsEndpoint  = flag.String("telemetry.endpoint", "/metrics", "Path under which to expose metric.")
	showVersion      = flag.Bool("version", false, "Print version information.")
)

const (
	namespace = "puppet_last_run_exporter"
)

type Exporter struct {
	mutex sync.Mutex

	resourcesChanged          *prometheus.Desc
	resourcesCorrectiveChange *prometheus.Desc
	resourcesFailed           *prometheus.Desc
	resourcesFailedRestart    *prometheus.Desc
	resourcesOutOfSync        *prometheus.Desc
	resourcesRestarted        *prometheus.Desc
	resourcesScheduled        *prometheus.Desc
	resourcesSkipped          *prometheus.Desc
	resourcesTotal            *prometheus.Desc

	timeCatalogApplication    *prometheus.Desc
	timeConfigRetrieval       *prometheus.Desc
	timeConvertCatalog        *prometheus.Desc
	timeExec                  *prometheus.Desc
	timeFactGeneration        *prometheus.Desc
	timeFilebucket            *prometheus.Desc
	timeNodeRetrieval         *prometheus.Desc
	timePluginsync            *prometheus.Desc
	timeTransactionEvaluation *prometheus.Desc
	timeTotal                 *prometheus.Desc
	timeLastRun               *prometheus.Desc
}

type T struct {
	Version struct {
		Config string `yaml: "version_config"`
		Puppet string `yaml: "version_puppet"`
	}

	Resources struct {
		Changed           float64 `yaml: "resources_changed"`
		Corrective_change float64 `yaml: "resouces_corrective_change"`
		Failed            float64
		Failed_to_restart float64
		Out_of_sync       float64
		Restarted         float64
		Scheduled         float64
		Skipped           float64
		Total             float64
	}

	Time struct {
		Anchor                 float64 `yaml: "timeAnchor"`
		Archive                float64 `yaml: "timeArchive"`
		Catalog_application    float64 `yaml: "timeCatalogApplication"`
		Config_retrieval       float64 `yaml: "timeConfigRetrieval"`
		Convert_catalog        float64 `yaml: "timeConverCatalog"`
		Exec                   float64 `yaml: "timeExec"`
		Fact_generation        float64 `yaml: "timeFactGeneration"`
		File                   float64 `yaml: "timeFile"`
		Filebucket             float64 `yaml: "timeFileBucket"`
		Group                  float64 `yaml: "timeGroup"`
		Node_retrieval         float64 `yaml: "timeNodeRetrieval"`
		Package                float64 `yaml: "timePackage"`
		Plugin_sync            float64 `yaml: "tomePluginSync"`
		Schedule               float64 `yaml: "timeSchedule"`
		Service                float64 `yaml: "timeService"`
		Total                  float64 `yaml: "timeTotal"`
		Transaction_evaluation float64 `yaml: "timeTransactionEvaluation"`
		User                   float64 `yaml: "timeUser"`
		Yumrepo                float64 `yaml: "timeYumrepo"`
		Last_run               float64 `yaml: "timeLastRunEpoch"`
	}
	Changes struct {
		Changes float64
		Total   float64
	}
	Events struct {
		Failure float64
		Success float64
		Total   float64
	}
}

func init() {
	prometheus.MustRegister(version.NewCollector("puppet_last_run_exporter"))
}

func NewSummaryExporter() *Exporter {
	return &Exporter{
		resourcesChanged: prometheus.NewDesc(prometheus.BuildFQName(namespace, "resources", "ResourcesChanged"),
			"Number of changed resources",
			nil,
			nil,
		),
		resourcesCorrectiveChange: prometheus.NewDesc(prometheus.BuildFQName(namespace, "resources", "ResourcesCorrectiveChange"),
			"Number of corrective changes",
			nil,
			nil,
		),
		resourcesFailed: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "ResourcesFailed"),
			"Number of failed resources",
			nil,
			nil,
		),
		resourcesFailedRestart: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "ResourcesFailedRestart"),
			"Number of resources failed to restart",
			nil,
			nil,
		),
		resourcesOutOfSync: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "ResourcesOutOfSync"),
			"Number of resources out of sync",
			nil,
			nil,
		),
		resourcesRestarted: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "ResourcesRestarted"),
			"Number of restarted resources",
			nil,
			nil,
		),
		resourcesScheduled: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "ResourcesScheduled"),
			"Number of scheduled resources",
			nil,
			nil,
		),
		resourcesSkipped: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "ResourcesSkipped"),
			"Number of skipped resources",
			nil,
			nil,
		),
		resourcesTotal: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "ResourcesTotal"),
			"Total number of resources",
			nil,
			nil,
		),
		timeCatalogApplication: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeCatalogApplication"),
			"Castalog application time",
			nil,
			nil,
		),
		timeConfigRetrieval: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeConfigRetrieval"),
			"Castalog application time",
			nil,
			nil,
		),
		timeConvertCatalog: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeConvertCatalog"),
			"Castalog conversion time",
			nil,
			nil,
		),
		timeFactGeneration: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeFactGeneration"),
			"Fact generation time",
			nil,
			nil,
		),
		timeFilebucket: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeFileBucket"),
			"Filebucket time",
			nil,
			nil,
		),
		timeNodeRetrieval: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeNodeRetrieval"),
			"Node retrieval time",
			nil,
			nil,
		),
		timePluginsync: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimePluginSync"),
			"Plugin sync time",
			nil,
			nil,
		),
		timeTransactionEvaluation: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeTransactionEvaluation"),
			"Transaction Evaluation time",
			nil,
			nil,
		),
		timeTotal: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeTotal"),
			"Total time",
			nil,
			nil,
		),
		timeLastRun: prometheus.NewDesc(prometheus.BuildFQName(namespace, "", "TimeLastRun"),
			"Last puppet run",
			nil,
			nil,
		),
	}

}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	if err := e.collect(ch); err != nil {
		log.Printf("Error scraping puppet_last_run_report: %s", err)
	}
	return
}

func (e *Exporter) collect(ch chan<- prometheus.Metric) error {

	dat, error := ioutil.ReadFile("./last_run_summary.yaml")
	if error != nil {
		log.Fatal(error)
	}

	var t T

	err := yaml.Unmarshal(dat, &t)
	if err != nil {
		panic(err)
	}

	ch <- prometheus.MustNewConstMetric(e.resourcesChanged, prometheus.GaugeValue, t.Resources.Changed)
	ch <- prometheus.MustNewConstMetric(e.resourcesCorrectiveChange, prometheus.GaugeValue, t.Resources.Corrective_change)
	ch <- prometheus.MustNewConstMetric(e.resourcesFailed, prometheus.GaugeValue, t.Resources.Failed)
	ch <- prometheus.MustNewConstMetric(e.resourcesFailedRestart, prometheus.GaugeValue, t.Resources.Failed_to_restart)
	ch <- prometheus.MustNewConstMetric(e.resourcesOutOfSync, prometheus.GaugeValue, t.Resources.Out_of_sync)
	ch <- prometheus.MustNewConstMetric(e.resourcesRestarted, prometheus.GaugeValue, t.Resources.Restarted)
	ch <- prometheus.MustNewConstMetric(e.resourcesScheduled, prometheus.GaugeValue, t.Resources.Scheduled)
	ch <- prometheus.MustNewConstMetric(e.resourcesSkipped, prometheus.GaugeValue, t.Resources.Skipped)
	ch <- prometheus.MustNewConstMetric(e.resourcesTotal, prometheus.GaugeValue, t.Resources.Total)
	ch <- prometheus.MustNewConstMetric(e.timeCatalogApplication, prometheus.GaugeValue, t.Time.Catalog_application)
	ch <- prometheus.MustNewConstMetric(e.timeConfigRetrieval, prometheus.GaugeValue, t.Time.Config_retrieval)
	ch <- prometheus.MustNewConstMetric(e.timeConvertCatalog, prometheus.GaugeValue, t.Time.Convert_catalog)
	ch <- prometheus.MustNewConstMetric(e.timeFactGeneration, prometheus.GaugeValue, t.Time.Fact_generation)
	ch <- prometheus.MustNewConstMetric(e.timeFilebucket, prometheus.GaugeValue, t.Time.Filebucket)
	ch <- prometheus.MustNewConstMetric(e.timeNodeRetrieval, prometheus.GaugeValue, t.Time.Node_retrieval)
	ch <- prometheus.MustNewConstMetric(e.timePluginsync, prometheus.GaugeValue, t.Time.Plugin_sync)
	ch <- prometheus.MustNewConstMetric(e.timeTransactionEvaluation, prometheus.GaugeValue, t.Time.Transaction_evaluation)
	ch <- prometheus.MustNewConstMetric(e.timeTotal, prometheus.GaugeValue, t.Time.Total)
	ch <- prometheus.MustNewConstMetric(e.timeLastRun, prometheus.CounterValue, t.Time.Last_run)
	return nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- e.resourcesChanged
	ch <- e.resourcesCorrectiveChange
	ch <- e.resourcesFailed
	ch <- e.resourcesFailedRestart
	ch <- e.resourcesOutOfSync
	ch <- e.resourcesRestarted
	ch <- e.resourcesScheduled
	ch <- e.resourcesSkipped
	ch <- e.resourcesTotal
	ch <- e.timeCatalogApplication
	ch <- e.timeConfigRetrieval
	ch <- e.timeFactGeneration
	ch <- e.timeFilebucket
	ch <- e.timeNodeRetrieval
	ch <- e.timePluginsync
	ch <- e.timeTransactionEvaluation
	ch <- e.timeTotal
	ch <- e.timeLastRun
}

func main() {

	flag.Parse()

	summaryExporter := NewSummaryExporter()
	prometheus.MustRegister(summaryExporter)

	log.Printf("Starting Server: %s", *listeningAddress)
	http.Handle(*metricsEndpoint, promhttp.Handler())
	log.Fatal(http.ListenAndServe(*listeningAddress, nil))

}
