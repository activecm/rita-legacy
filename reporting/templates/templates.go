package templates

import "html/template"

//ReportingInfo fills the templates listed in html/template
type ReportingInfo struct {
	DB     string
	Writer template.HTML
}

var ocmdevImg = "<img src=\"data:image/png;base64,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\" alt=\"Offensive Countermeasures\" style=\"width:90px; float:left\" />"

var dbHeader = `
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="../style.css">
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
	` + ocmdevImg + `
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a href="index.html">Viewing: {{.DB}}</a></li>
  <li><a href="beacons.html">Beacons</a></li>
	<li><a href="dns.html">DNS</a></li>
  <li><a href="bl-source-ips.html">BL Source IPs</a></li>
	<li><a href="bl-dest-ips.html">BL Dest. IPs</a></li>
	<li><a href="bl-hostnames.html">BL Hostnames</a></li>
	<li><a href="bl-urls.html">BL URLs</a></li>
	<li><a href="scans.html">Scans</a></li>
	<li><a href="long-conns.html">Long Connections</a></li>
	<li><a href="long-urls.html">Long URLs</a></li>
	<li><a href="useragents.html">User Agents</a></li>
	<li style="float:right">
    <a href="https://github.com/ocmdev/rita" target="_blank">RITA on
		<img src="../github.svg" title="Icon made by Dave Gandy from www.flaticon.com" id="github">
    </a>
  </li>
</ul>
`

var homeHeader = `
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="./style.css">
</head>
<ul>
    <a href="http://offensivecountermeasures.com/" target="_blank">
		` + ocmdevImg + `
    </a>
  <li><a href="./index.html">RITA</a></li>
  <li style="float:right">
    <a href="https://github.com/ocmdev/rita" target="_blank">RITA on
		<img src="./github.svg" title="Icon made by Dave Gandy from www.flaticon.com" id="github">
    </a>
  </li>
</ul>
`

// ScansTempl is our scans html template
var ScansTempl = dbHeader + `
<div class="container">
  <table>
    <tr><th>Source</th><th>Destination</th><th>Port Count</th><th>Port Set</th></tr>
      {{.Writer}}
  </table>
</div>
`

// Hometempl is our home template html
var Hometempl = homeHeader + `
<p>
  <div class="info">To view individual databases, click on any of the links below.</div>
  <div class="vertical-menu">
    {{range .}}
      <a href="{{.}}/index.html">{{.}}</a>
    {{end}}
  </div>
</p>
`

// DNStempl is our dns page template
var DNStempl = dbHeader + `
<div class="container">
  <table>
    <tr><th>Subdomain</th><th>Visited</th><th>Domain</th><tr>
    {{.Writer}}
  </table>
</div>
`

// DBhometempl is our database home template for each directory
var DBhometempl = dbHeader + `
<p>
  <div class="info">To view results, click on any of the links above.</div>
</p>
`

// BeaconsTempl is our beacons html template
var BeaconsTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>Score</th><th>Source</th><th>Destination</th><th>Connections</th><th>Avg. Bytes</th><th>
	Intvl. Range</th><th>Size Range</th><th>Intvl. Mode</th><th>Size Mode</th><th>Intvl. Mode Count</th>
	<th>Size Mode Count</th><th>Intvl. Skew</th><th>Size Skew</th><th>Intvl. Dispersion</th><th>Size Dispersion
	</th><th>TS Duration</tr>
      {{.Writer}}
  </table>
</div>
`

// BLSourceIPTempl is our blacklisted source ip html template
var BLSourceIPTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>IP</th><th>Connections</th><th>Unique Connections</th><th>Total Bytes</th><th>Lists</th><th>Destinations</th><tr>
    {{.Writer}}
  </table>
</div>
`

// BLDestIPTempl is our blacklisted destination ip html template
var BLDestIPTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>IP</th><th>Connections</th><th>Unique Connections</th><th>Total Bytes</th><th>Lists</th><th>Sources</th><tr>
    {{.Writer}}
  </table>
</div>
`

// BLHostnameTempl is our blacklisted hostname html template
var BLHostnameTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>Hostname</th><th>Connections</th><th>Unique Connections</th><th>Total Bytes</th><th>Lists</th><th>Sources</th><tr>
    {{.Writer}}
  </table>
</div>
`

// BLURLTempl is our blacklisted url html template
var BLURLTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>Host</th><th>Resource</th><th>Connections</th><th>Unique Connections</th><th>Total Bytes</th><th>Lists</th><th>Sources</th><tr>
    {{.Writer}}
  </table>
</div>
`

// LongConnsTempl is our long connections html template
var LongConnsTempl = dbHeader + `
<div class="container">
  <table>
	<tr><th>Source</th><th>Source Port</th><th>Destination</th><th>Destination Port</th><th>Duration</th><th>Protocol</th></tr>
	  {{.Writer}}
	</table>
</div>
`

// LongURLsTempl is our long urls html template
var LongURLsTempl = dbHeader + `
<div class="container">
  <table>
	<tr><th>URL</th><th>URI</th><th>Length</th><th>Times Visited</th></tr>
	  {{.Writer}}
	</table>
</div>
`

// UserAgentsTempl is our user agents html template
var UserAgentsTempl = dbHeader + `
<div class="container">
  <table>
	<tr><th>User Agent</th><th>Times Used</th></tr>
	  {{.Writer}}
	</table>
</div>
`
