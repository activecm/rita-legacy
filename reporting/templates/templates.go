package templates

import "html/template"

//ReportingInfo fills the templates listed in html/template
type ReportingInfo struct {
	DB     string
	Writer template.HTML
}

var dbHeader = `
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<link rel="stylesheet" type="text/css" href="../style.css">
<script src="https://use.fontawesome.com/96648b06fb.js"></script>
</head>

<ul>
  <a href="http://offensivecountermeasures.com/" target="_blank">
    <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
    style="width:90px; float:left" />
  </a>
  <li><a href="../index.html">RITA</a></li>
  <li><a href="index.html">Viewing: {{.DB}}</a></li>
  <li><a href="beacons.html">Beacons</a></li>
  <li><a href="blacklisted.html">Blacklisted</a></li>
  <li><a href="dns.html">DNS</a></li>
  <li><a href="scans.html">Scans</a></li>
  <li><a href="long-conns.html">Long Connections</a></li>
	<li><a href="long-urls.html">Long URLs</a></li>
	<li><a href="useragents.html">User Agents</a></li>
  <li style="float:right">
    <a href="https://github.com/ocmdev/rita" target="_blank">RITA on
      <i class="fa fa-github fa-lg" aria-hidden="true" alt="GitHub"></i>
    </a>
  </li>
</ul>
`

var homeHeader = `
<head>
<meta content="text/html;charset=utf-8" http-equiv="Content-Type">
<meta content="utf-8" http-equiv="encoding">
<script src="https://use.fontawesome.com/96648b06fb.js"></script>
<link rel="stylesheet" type="text/css" href="./style.css">
</head>
<ul>
    <a href="http://offensivecountermeasures.com/" target="_blank">
      <img src="http://45.33.27.128/wp-content/uploads/2016/02/OCM-logo-022416.png" alt="Offensive Countermeasures"
      style="width:90px; float:left" />
    </a>
  <li><a href="./index.html">RITA</a></li>
  <li style="float:right">
    <a href="https://github.com/ocmdev/rita" target="_blank">RITA on
      <i class="fa fa-github fa-lg" aria-hidden="true" alt="GitHub"></i>
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
  <tr><th>TS score</th><th>Source</th><th>Destination</th><th>Connections</th><th>Avg. Bytes</th><th>
	Intvl. Range</th><th>Intvl. Mode</th><th>Intvl. Mode Count</th><th>
	Intvl. Skew</th><th>Intvl. Dispersion</th><th>TS Duration</tr>
      {{.Writer}}
  </table>
</div>
`

// BlacklistedTempl is our beacons html template
var BlacklistedTempl = dbHeader + `
<div class="container">
  <table>
  <tr><th>Destination</th><th>Score</th><th>Source(s)</th><tr>
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
