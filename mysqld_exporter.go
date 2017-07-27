package main

import (
	"database/sql"
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"path"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/log"
	"github.com/prometheus/common/version"
	"gopkg.in/ini.v1"

	"github.com/prometheus/mysqld_exporter/collector"
)

var (
	showVersion = flag.Bool(
		"version", false,
		"Print version information.",
	)
	listenAddress = flag.String(
		"web.listen-address", ":9104",
		"Address to listen on for web interface and telemetry.",
	)
	metricPath = flag.String(
		"web.telemetry-path", "/metrics",
		"Path under which to expose metrics.",
	)
	configMycnf = flag.String(
		"config.my-cnf", path.Join(os.Getenv("HOME"), ".my.cnf"),
		"Path to .my.cnf file to read MySQL credentials from.",
	)
	slowLogFilter = flag.Bool(
		"log_slow_filter", false,
		"Add a log_slow_filter to avoid exessive MySQL slow logging.  NOTE: Not supported by Oracle MySQL.",
	)
	collectProcesslist = flag.Bool(
		"collect.info_schema.processlist", false,
		"Collect current thread state counts from the information_schema.processlist",
	)
	collectTableSchema = flag.Bool(
		"collect.info_schema.tables", true,
		"Collect metrics from information_schema.tables",
	)
	collectInnodbTablespaces = flag.Bool(
		"collect.info_schema.innodb_tablespaces", false,
		"Collect metrics from information_schema.innodb_sys_tablespaces",
	)
	innodbMetrics = flag.Bool(
		"collect.info_schema.innodb_metrics", false,
		"Collect metrics from information_schema.innodb_metrics",
	)
	tableSchemaAutoAnalyze = flag.Bool(
		"collect.info_schema.tables.auto_analyze", false,
		"Automatically run ANALYZE TABLE to update stats",
	)
	tableSchemaAutoAnalyzeDuration = flag.Duration(
		"collect.info_schema.tables.auto_analyze.min_duration", 5*time.Minute,
		"Minimum seconds between automatic ANALYZE TABLE calls",
	)
	collectGlobalStatus = flag.Bool(
		"collect.global_status", true,
		"Collect from SHOW GLOBAL STATUS",
	)
	collectGlobalVariables = flag.Bool(
		"collect.global_variables", true,
		"Collect from SHOW GLOBAL VARIABLES",
	)
	collectSlaveStatus = flag.Bool(
		"collect.slave_status", true,
		"Collect from SHOW SLAVE STATUS",
	)
	collectAutoIncrementColumns = flag.Bool(
		"collect.auto_increment.columns", false,
		"Collect auto_increment columns and max values from information_schema",
	)
	collectBinlogSize = flag.Bool(
		"collect.binlog_size", false,
		"Collect the current size of all registered binlog files",
	)
	collectPerfTableIOWaits = flag.Bool(
		"collect.perf_schema.tableiowaits", false,
		"Collect metrics from performance_schema.table_io_waits_summary_by_table",
	)
	collectPerfIndexIOWaits = flag.Bool(
		"collect.perf_schema.indexiowaits", false,
		"Collect metrics from performance_schema.table_io_waits_summary_by_index_usage",
	)
	collectPerfTableLockWaits = flag.Bool(
		"collect.perf_schema.tablelocks", false,
		"Collect metrics from performance_schema.table_lock_waits_summary_by_table",
	)
	collectPerfEventsStatements = flag.Bool(
		"collect.perf_schema.eventsstatements", false,
		"Collect metrics from performance_schema.events_statements_summary_by_digest",
	)
	collectPerfEventsWaits = flag.Bool(
		"collect.perf_schema.eventswaits", false,
		"Collect metrics from performance_schema.events_waits_summary_global_by_event_name",
	)
	collectPerfFileEvents = flag.Bool(
		"collect.perf_schema.file_events", false,
		"Collect metrics from performance_schema.file_summary_by_event_name",
	)
	collectPerfFileInstances = flag.Bool(
		"collect.perf_schema.file_instances", false,
		"Collect metrics from performance_schema.file_summary_by_instance",
	)
	collectUserStat = flag.Bool("collect.info_schema.userstats", false,
		"If running with userstat=1, set to true to collect user statistics",
	)
	collectClientStat = flag.Bool("collect.info_schema.clientstats", false,
		"If running with userstat=1, set to true to collect client statistics",
	)
	collectTableStat = flag.Bool("collect.info_schema.tablestats", false,
		"If running with userstat=1, set to true to collect table statistics",
	)
	collectQueryResponseTime = flag.Bool("collect.info_schema.query_response_time", false,
		"Collect query response time distribution if query_response_time_stats is ON.",
	)
	collectEngineTokudbStatus = flag.Bool("collect.engine_tokudb_status", false,
		"Collect from SHOW ENGINE TOKUDB STATUS",
	)
	collectEngineInnodbStatus = flag.Bool("collect.engine_innodb_status", false,
		"Collect from SHOW ENGINE INNODB STATUS",
	)
	collectHeartbeat = flag.Bool(
		"collect.heartbeat", false,
		"Collect from heartbeat",
	)
	collectHeartbeatDatabase = flag.String(
		"collect.heartbeat.database", "heartbeat",
		"Database from where to collect heartbeat data",
	)
	collectHeartbeatTable = flag.String(
		"collect.heartbeat.table", "heartbeat",
		"Table from where to collect heartbeat data",
	)
)

// Metric name parts.
const (
	// Namespace for all metrics.
	namespace = "mysql"
	// Subsystem(s).
	exporter = "exporter"
)

// SQL Queries.
const (
<<<<<<< HEAD
	globalStatusQuery          = `SHOW GLOBAL STATUS`
	globalVariablesQuery       = `SHOW GLOBAL VARIABLES`
	slaveStatusQuery           = `SHOW SLAVE STATUS`
	binlogQuery                = `SHOW BINARY LOGS`
	infoSchemaProcesslistQuery = `
		SELECT COALESCE(command,''),COALESCE(state,''),count(*)
		FROM information_schema.processlist
		WHERE ID != connection_id()
		GROUP BY command,state
		ORDER BY null`
	infoSchemaAutoIncrementQuery = `
		SELECT table_schema, table_name, column_name, auto_increment,
		  pow(2, case data_type
		    when 'tinyint'   then 7
		    when 'smallint'  then 15
		    when 'mediumint' then 23
		    when 'int'       then 31
		    when 'bigint'    then 63
		    end+(column_type like '% unsigned'))-1 as max_int
		  FROM information_schema.tables t
		  JOIN information_schema.columns c USING (table_schema,table_name)
		  WHERE c.extra = 'auto_increment' AND t.auto_increment IS NOT NULL
		`
	perfTableIOWaitsQuery = `
		SELECT OBJECT_SCHEMA, OBJECT_NAME, COUNT_FETCH, COUNT_INSERT, COUNT_UPDATE, COUNT_DELETE,
		  SUM_TIMER_FETCH, SUM_TIMER_INSERT, SUM_TIMER_UPDATE, SUM_TIMER_DELETE
		  FROM performance_schema.table_io_waits_summary_by_table
		  WHERE OBJECT_SCHEMA NOT IN ('mysql', 'performance_schema')
		`
	perfIndexIOWaitsQuery = `
		SELECT OBJECT_SCHEMA, OBJECT_NAME, ifnull(INDEX_NAME, 'NONE') as INDEX_NAME,
		  COUNT_FETCH, COUNT_INSERT, COUNT_UPDATE, COUNT_DELETE,
		  SUM_TIMER_FETCH, SUM_TIMER_INSERT, SUM_TIMER_UPDATE, SUM_TIMER_DELETE
		  FROM performance_schema.table_io_waits_summary_by_index_usage
		  WHERE OBJECT_SCHEMA NOT IN ('mysql', 'performance_schema')
		`
	perfTableLockWaitsQuery = `
		SELECT
		    OBJECT_SCHEMA,
		    OBJECT_NAME,
		    COUNT_READ_NORMAL,
		    COUNT_READ_WITH_SHARED_LOCKS,
		    COUNT_READ_HIGH_PRIORITY,
		    COUNT_READ_NO_INSERT,
		    COUNT_READ_EXTERNAL,
		    COUNT_WRITE_ALLOW_WRITE,
		    COUNT_WRITE_CONCURRENT_INSERT,
		    COUNT_WRITE_DELAYED,
		    COUNT_WRITE_LOW_PRIORITY,
		    COUNT_WRITE_NORMAL,
		    COUNT_WRITE_EXTERNAL,
		    SUM_TIMER_READ_NORMAL,
		    SUM_TIMER_READ_WITH_SHARED_LOCKS,
		    SUM_TIMER_READ_HIGH_PRIORITY,
		    SUM_TIMER_READ_NO_INSERT,
		    SUM_TIMER_READ_EXTERNAL,
		    SUM_TIMER_WRITE_ALLOW_WRITE,
		    SUM_TIMER_WRITE_CONCURRENT_INSERT,
		    SUM_TIMER_WRITE_DELAYED,
		    SUM_TIMER_WRITE_LOW_PRIORITY,
		    SUM_TIMER_WRITE_NORMAL,
		    SUM_TIMER_WRITE_EXTERNAL
		  FROM performance_schema.table_lock_waits_summary_by_table
		  WHERE OBJECT_SCHEMA NOT IN ('mysql', 'performance_schema', 'information_schema')
		`
	perfEventsStatementsQuery = `
		SELECT
		    ifnull(SCHEMA_NAME, 'NONE') as SCHEMA_NAME,
		    DIGEST,
		    LEFT(DIGEST_TEXT, %d) as DIGEST_TEXT,
		    COUNT_STAR,
		    SUM_TIMER_WAIT,
		    SUM_ERRORS,
		    SUM_WARNINGS,
		    SUM_ROWS_AFFECTED,
		    SUM_ROWS_SENT,
		    SUM_ROWS_EXAMINED,
		    SUM_CREATED_TMP_DISK_TABLES,
		    SUM_CREATED_TMP_TABLES,
		    SUM_SORT_MERGE_PASSES,
		    SUM_SORT_ROWS,
		    SUM_NO_INDEX_USED
		  FROM performance_schema.events_statements_summary_by_digest
		  WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'information_schema')
		    AND last_seen > DATE_SUB(NOW(), INTERVAL %d SECOND)
		  ORDER BY SUM_TIMER_WAIT DESC
		  LIMIT %d
		`
	perfEventsWaitsQuery = `
		SELECT EVENT_NAME, COUNT_STAR, SUM_TIMER_WAIT
		  FROM performance_schema.events_waits_summary_global_by_event_name
		`
	perfFileEventsQuery = `
		SELECT
		  EVENT_NAME,
		  COUNT_READ, SUM_TIMER_READ, SUM_NUMBER_OF_BYTES_READ,
		  COUNT_WRITE, SUM_TIMER_WRITE, SUM_NUMBER_OF_BYTES_WRITE,
		  COUNT_MISC, SUM_TIMER_MISC
		  FROM performance_schema.file_summary_by_event_name
		`
	userStatQuery  = `SELECT * FROM information_schema.USER_STATISTICS`
	tableStatQuery = `
		SELECT
			TABLE_SCHEMA,
			TABLE_NAME,
			ROWS_READ,
			ROWS_CHANGED,
			ROWS_CHANGED_X_INDEXES
		  FROM information_schema.TABLE_STATISTICS
		`
	tableSchemaQuery = `
		SELECT
		    TABLE_SCHEMA,
		    TABLE_NAME,
		    TABLE_TYPE,
		    ifnull(ENGINE, 'NONE') as ENGINE,
		    ifnull(VERSION, '0') as VERSION,
		    ifnull(ROW_FORMAT, 'NONE') as ROW_FORMAT,
		    ifnull(TABLE_ROWS, '0') as TABLE_ROWS,
		    ifnull(DATA_LENGTH, '0') as DATA_LENGTH,
		    ifnull(INDEX_LENGTH, '0') as INDEX_LENGTH,
		    ifnull(DATA_FREE, '0') as DATA_FREE,
		    ifnull(CREATE_OPTIONS, 'NONE') as CREATE_OPTIONS
		  FROM information_schema.tables
		  WHERE TABLE_SCHEMA = '%s'
		`
	dbListQuery = `
		SELECT
		    SCHEMA_NAME
		  FROM information_schema.schemata
		  WHERE SCHEMA_NAME NOT IN ('mysql', 'performance_schema', 'information_schema')
		`
	tableListQuery = `
		SELECT
		    TABLE_NAME
		  FROM information_schema.tables
		  WHERE TABLE_SCHEMA = '%s'
		    AND TABLE_TYPE = 'BASE TABLE'
		    AND ENGINE IS NOT NULL
		`
	analyzeTableQuery = `ANALYZE TABLE %s.%s`
=======
	sessionSettingsQuery = `SET SESSION log_slow_filter = 'tmp_table_on_disk,filesort_on_disk'`
	upQuery              = `SELECT 1`
>>>>>>> e755b01ea4189bbc160027ffb8f2276f3b02a236
)

// landingPage contains the HTML served at '/'.
// TODO: Make this nicer and more informative.
var landingPage = []byte(`<html>
<head><title>MySQLd exporter</title></head>
<body>
<h1>MySQLd exporter</h1>
<p><a href='` + *metricPath + `'>Metrics</a></p>
</body>
</html>
`)

// Metric descriptors.
var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, exporter, "collector_duration_seconds"),
		"Collector time duration.",
		[]string{"collector"}, nil,
	)
)

// Exporter collects MySQL metrics. It implements prometheus.Collector.
type Exporter struct {
	dsn          string
	error        prometheus.Gauge
	totalScrapes prometheus.Counter
	scrapeErrors *prometheus.CounterVec
	mysqldUp     prometheus.Gauge
}

// Last run tracking variable for scrapeTableSchema().
var tableSchemaLastRun = map[string]time.Time{}

// NewExporter returns a new MySQL exporter for the provided DSN.
func NewExporter(dsn string) *Exporter {
	return &Exporter{
		dsn: dsn,
		totalScrapes: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "scrapes_total",
			Help:      "Total number of times MySQL was scraped for metrics.",
		}),
		scrapeErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "scrape_errors_total",
			Help:      "Total number of times an error occurred scraping a MySQL.",
		}, []string{"collector"}),
		error: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: exporter,
			Name:      "last_scrape_error",
			Help:      "Whether the last scrape of metrics from MySQL resulted in an error (1 for error, 0 for success).",
		}),
		mysqldUp: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "up",
			Help:      "Whether the MySQL server is up.",
		}),
	}
}

// Describe implements prometheus.Collector.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	// We cannot know in advance what metrics the exporter will generate
	// from MySQL. So we use the poor man's describe method: Run a collect
	// and send the descriptors of all the collected metrics. The problem
	// here is that we need to connect to the MySQL DB. If it is currently
	// unavailable, the descriptors will be incomplete. Since this is a
	// stand-alone exporter and not used as a library within other code
	// implementing additional metrics, the worst that can happen is that we
	// don't detect inconsistent metrics created by this exporter
	// itself. Also, a change in the monitored MySQL instance may change the
	// exported metrics during the runtime of the exporter.

	metricCh := make(chan prometheus.Metric)
	doneCh := make(chan struct{})

	go func() {
		for m := range metricCh {
			ch <- m.Desc()
		}
		close(doneCh)
	}()

	e.Collect(metricCh)
	close(metricCh)
	<-doneCh
}

// Collect implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.scrape(ch)

	ch <- e.totalScrapes
	ch <- e.error
	e.scrapeErrors.Collect(ch)
	ch <- e.mysqldUp
}

func (e *Exporter) scrape(ch chan<- prometheus.Metric) {
	e.totalScrapes.Inc()
	var err error

	scrapeTime := time.Now()
	db, err := sql.Open("mysql", e.dsn)
	if err != nil {
		log.Errorln("Error opening connection to database:", err)
		return
	}
	defer db.Close()

	// By design exporter should use maximum one connection per request.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	// Set max lifetime for a connection.
	db.SetConnMaxLifetime(1 * time.Minute)

	isUpRows, err := db.Query(upQuery)
	if err != nil {
		log.Errorln("Error pinging mysqld:", err)
		e.mysqldUp.Set(0)
		return
	}
	isUpRows.Close()

	e.mysqldUp.Set(1)

	if *slowLogFilter {
		sessionSettingsRows, err := db.Query(sessionSettingsQuery)
		if err != nil {
			log.Errorln("Error setting log_slow_filter:", err)
			return
		}
		sessionSettingsRows.Close()
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "connection")

	if *collectGlobalStatus {
		scrapeTime = time.Now()
		if err = collector.ScrapeGlobalStatus(db, ch); err != nil {
			log.Errorln("Error scraping for collect.global_status:", err)
			e.scrapeErrors.WithLabelValues("collect.global_status").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.global_status")
	}
	if *collectGlobalVariables {
		scrapeTime = time.Now()
		if err = collector.ScrapeGlobalVariables(db, ch); err != nil {
			log.Errorln("Error scraping for collect.global_variables:", err)
			e.scrapeErrors.WithLabelValues("collect.global_variables").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.global_variables")
	}
	if *collectSlaveStatus {
		scrapeTime = time.Now()
		if err = collector.ScrapeSlaveStatus(db, ch); err != nil {
			log.Errorln("Error scraping for collect.slave_status:", err)
			e.scrapeErrors.WithLabelValues("collect.slave_status").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.slave_status")
	}
	if *collectProcesslist {
		scrapeTime = time.Now()
		if err = collector.ScrapeProcesslist(db, ch); err != nil {
			log.Errorln("Error scraping for collect.info_schema.processlist:", err)
			e.scrapeErrors.WithLabelValues("collect.info_schema.processlist").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.info_schema.processlist")
	}
	if *collectTableSchema {
		scrapeTime = time.Now()
		if err = collector.ScrapeTableSchema(db, ch); err != nil {
			log.Errorln("Error scraping for collect.info_schema.tables:", err)
			e.scrapeErrors.WithLabelValues("collect.info_schema.tables").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.info_schema.tables")
	}
	if *collectInnodbTablespaces {
		scrapeTime = time.Now()
		if err = collector.ScrapeInfoSchemaInnodbTablespaces(db, ch); err != nil {
			log.Errorln("Error scraping for collect.info_schema.innodb_sys_tablespaces:", err)
			e.scrapeErrors.WithLabelValues("collect.info_schema.innodb_sys_tablespaces").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.info_schema.innodb_sys_tablespaces")
	}
	if *innodbMetrics {
		if err = collector.ScrapeInnodbMetrics(db, ch); err != nil {
			log.Errorln("Error scraping for collect.info_schema.innodb_metrics:", err)
			e.scrapeErrors.WithLabelValues("collect.info_schema.innodb_metrics").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.info_schema.innodb_metrics")
	}
	if *collectAutoIncrementColumns {
		scrapeTime = time.Now()
		if err = collector.ScrapeAutoIncrementColumns(db, ch); err != nil {
			log.Errorln("Error scraping for collect.auto_increment.columns:", err)
			e.scrapeErrors.WithLabelValues("collect.auto_increment.columns").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.auto_increment.columns")
	}
	if *collectBinlogSize {
		scrapeTime = time.Now()
		if err = collector.ScrapeBinlogSize(db, ch); err != nil {
			log.Errorln("Error scraping for collect.binlog_size:", err)
			e.scrapeErrors.WithLabelValues("collect.binlog_size").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.binlog_size")
	}
	if *collectPerfTableIOWaits {
		scrapeTime = time.Now()
		if err = collector.ScrapePerfTableIOWaits(db, ch); err != nil {
			log.Errorln("Error scraping for collect.perf_schema.tableiowaits:", err)
			e.scrapeErrors.WithLabelValues("collect.perf_schema.tableiowaits").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.perf_schema.tableiowaits")
	}
	if *collectPerfIndexIOWaits {
		scrapeTime = time.Now()
		if err = collector.ScrapePerfIndexIOWaits(db, ch); err != nil {
			log.Errorln("Error scraping for collect.perf_schema.indexiowaits:", err)
			e.scrapeErrors.WithLabelValues("collect.perf_schema.indexiowaits").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.perf_schema.indexiowaits")
	}
	if *collectPerfTableLockWaits {
		scrapeTime = time.Now()
		if err = collector.ScrapePerfTableLockWaits(db, ch); err != nil {
			log.Errorln("Error scraping for collect.perf_schema.tablelocks:", err)
			e.scrapeErrors.WithLabelValues("collect.perf_schema.tablelocks").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.perf_schema.tablelocks")
	}
	if *collectPerfEventsStatements {
		scrapeTime = time.Now()
		if err = collector.ScrapePerfEventsStatements(db, ch); err != nil {
			log.Errorln("Error scraping for collect.perf_schema.eventsstatements:", err)
			e.scrapeErrors.WithLabelValues("collect.perf_schema.eventsstatements").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.perf_schema.eventsstatements")
	}
	if *collectPerfEventsWaits {
		scrapeTime = time.Now()
		if err = collector.ScrapePerfEventsWaits(db, ch); err != nil {
			log.Errorln("Error scraping for collect.perf_schema.eventswaits:", err)
			e.scrapeErrors.WithLabelValues("collect.perf_schema.eventswaits").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.perf_schema.eventswaits")
	}
	if *collectPerfFileEvents {
		scrapeTime = time.Now()
		if err = collector.ScrapePerfFileEvents(db, ch); err != nil {
			log.Errorln("Error scraping for collect.perf_schema.file_events:", err)
			e.scrapeErrors.WithLabelValues("collect.perf_schema.file_events").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.perf_schema.file_events")
	}
	if *collectPerfFileInstances {
		scrapeTime = time.Now()
		if err = collector.ScrapePerfFileInstances(db, ch); err != nil {
			log.Errorln("Error scraping for collect.perf_schema.file_instances:", err)
			e.scrapeErrors.WithLabelValues("collect.perf_schema.file_instances").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.perf_schema.file_instances")
	}
	if *collectUserStat {
		scrapeTime = time.Now()
		if err = collector.ScrapeUserStat(db, ch); err != nil {
			log.Errorln("Error scraping for collect.info_schema.userstats:", err)
			e.scrapeErrors.WithLabelValues("collect.info_schema.userstats").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.info_schema.userstats")
	}
	if *collectClientStat {
		scrapeTime = time.Now()
		if err = collector.ScrapeClientStat(db, ch); err != nil {
			log.Errorln("Error scraping for collect.info_schema.clientstats:", err)
			e.scrapeErrors.WithLabelValues("collect.info_schema.clientstats").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.info_schema.clientstats")
	}
	if *collectTableStat {
		scrapeTime = time.Now()
		if err = collector.ScrapeTableStat(db, ch); err != nil {
			log.Errorln("Error scraping for collect.info_schema.tablestats:", err)
			e.scrapeErrors.WithLabelValues("collect.info_schema.tablestats").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.info_schema.tablestats")
	}
	if *collectQueryResponseTime {
		scrapeTime = time.Now()
		if err = collector.ScrapeQueryResponseTime(db, ch); err != nil {
			log.Errorln("Error scraping for collect.info_schema.query_response_time:", err)
			e.scrapeErrors.WithLabelValues("collect.info_schema.query_response_time").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.info_schema.query_response_time")
	}
	if *collectEngineTokudbStatus {
		scrapeTime = time.Now()
		if err = collector.ScrapeEngineTokudbStatus(db, ch); err != nil {
			log.Errorln("Error scraping for collect.engine_tokudb_status:", err)
			e.scrapeErrors.WithLabelValues("collect.engine_tokudb_status").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.engine_tokudb_status")
	}
	if *collectEngineInnodbStatus {
		scrapeTime = time.Now()
		if err = collector.ScrapeEngineInnodbStatus(db, ch); err != nil {
			log.Errorln("Error scraping for collect.engine_innodb_status:", err)
			e.scrapeErrors.WithLabelValues("collect.engine_innodb_status").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.engine_innodb_status")
	}
	if *collectHeartbeat {
		scrapeTime = time.Now()
		if err = collector.ScrapeHeartbeat(db, ch, collectHeartbeatDatabase, collectHeartbeatTable); err != nil {
			log.Errorln("Error scraping for collect.heartbeat:", err)
			e.scrapeErrors.WithLabelValues("collect.heartbeat").Inc()
		}
		ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, time.Since(scrapeTime).Seconds(), "collect.heartbeat")
	}
}

func parseMycnf(config interface{}) (string, error) {
	var dsn string
	cfg, err := ini.Load(config)
	if err != nil {
		return dsn, fmt.Errorf("failed reading ini file: %s", err)
	}
	user := cfg.Section("client").Key("user").String()
	password := cfg.Section("client").Key("password").String()
	if (user == "") || (password == "") {
		return dsn, fmt.Errorf("no user or password specified under [client] in %s", config)
	}
	host := cfg.Section("client").Key("host").MustString("localhost")
	port := cfg.Section("client").Key("port").MustUint(3306)
	socket := cfg.Section("client").Key("socket").String()
	if socket != "" {
		dsn = fmt.Sprintf("%s:%s@unix(%s)/", user, password, socket)
	} else {
<<<<<<< HEAD
		dbList = strings.Split(*tableSchemaDatabases, ",")
	}

	for _, database := range dbList {
		random := 1 + rand.Float64() * 0.25
		nextRun := time.Duration(float64(*tableSchemaAutoAnalyzeDuration)*random)
		if *tableSchemaAutoAnalyze && time.Since(tableSchemaLastRun[database]) > nextRun {
			tablesRows, err := db.Query(fmt.Sprintf(tableListQuery, database))
			tableSchemaLastRun[database] = time.Now()
			log.Debugf("Ran AutoAnalyze on %s", database)
			if err != nil {
				return err
			}
			defer tablesRows.Close()

			var tableName string

			for tablesRows.Next() {
				err = tablesRows.Scan(&tableName)
				if err != nil {
					return err
				}
				_, err := db.Query(fmt.Sprintf(analyzeTableQuery, database, tableName))
				if err != nil {
					return err
				}
			}
		}
		tableSchemaRows, err := db.Query(fmt.Sprintf(tableSchemaQuery, database))
		if err != nil {
			return err
		}
		defer tableSchemaRows.Close()

		var (
			tableSchema   string
			tableName     string
			tableType     string
			engine        string
			version       uint64
			rowFormat     string
			tableRows     uint64
			dataLength    uint64
			indexLength   uint64
			dataFree      uint64
			createOptions string
		)

		for tableSchemaRows.Next() {
			err = tableSchemaRows.Scan(
				&tableSchema,
				&tableName,
				&tableType,
				&engine,
				&version,
				&rowFormat,
				&tableRows,
				&dataLength,
				&indexLength,
				&dataFree,
				&createOptions,
			)
			if err != nil {
				return err
			}
			ch <- prometheus.MustNewConstMetric(
				infoSchemaTablesVersionDesc, prometheus.GaugeValue, float64(version),
				tableSchema, tableName, tableType, engine, rowFormat, createOptions,
			)
			ch <- prometheus.MustNewConstMetric(
				infoSchemaTablesRowsDesc, prometheus.GaugeValue, float64(tableRows),
				tableSchema, tableName,
			)
			ch <- prometheus.MustNewConstMetric(
				infoSchemaTablesSizeDesc, prometheus.GaugeValue, float64(dataLength),
				tableSchema, tableName, "data_length",
			)
			ch <- prometheus.MustNewConstMetric(
				infoSchemaTablesSizeDesc, prometheus.GaugeValue, float64(indexLength),
				tableSchema, tableName, "index_length",
			)
			ch <- prometheus.MustNewConstMetric(
				infoSchemaTablesSizeDesc, prometheus.GaugeValue, float64(dataFree),
				tableSchema, tableName, "data_free",
			)
		}
=======
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
>>>>>>> e755b01ea4189bbc160027ffb8f2276f3b02a236
	}
	log.Debugln(dsn)
	return dsn, nil
}

func init() {
	prometheus.MustRegister(version.NewCollector("mysqld_exporter"))
}

func main() {
	flag.Parse()

	if *showVersion {
		fmt.Fprintln(os.Stdout, version.Print("mysqld_exporter"))
		os.Exit(0)
	}

	log.Infoln("Starting mysqld_exporter", version.Info())
	log.Infoln("Build context", version.BuildContext())

	dsn := os.Getenv("DATA_SOURCE_NAME")
	if len(dsn) == 0 {
		var err error
		if dsn, err = parseMycnf(*configMycnf); err != nil {
			log.Fatal(err)
		}
	}

	exporter := NewExporter(dsn)
	prometheus.MustRegister(exporter)

	http.Handle(*metricPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(landingPage)
	})

	log.Infoln("Listening on", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
