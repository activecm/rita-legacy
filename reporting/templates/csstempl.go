package templates

// CSStempl is our css template sheet
var CSStempl = []byte(`p {
  margin-bottom: 1.625em;
  font-family: 'Lucida Sans', Arial, sans-serif;
}

p {
  font-family: 'Lucida Sans', Arial, sans-serif;
  text-indent: 30px;
}

h1 {
  color: #000;
  font-family: 'Lato', sans-serif;
  font-size: 32px;
  font-weight: 300;
  line-height: 58px;
  margin: 0 0 58px;
  text-indent: 30px;
}

ul {
  list-style-type: none;
  margin: 0;
  padding: 0;
  overflow: hidden;
  background-color: #000;
  font-family: "Arial", Helvetica, sans-serif;
}

li {
  float: left;
  border-right: 1px solid #bbb;
}

li:last-child {
  border-right: none;
}

li a {
  display: block;
  color: white;
  text-align: center;
  padding: 14px 16px;
  text-decoration: none;
}

div {
  color: #adb7bd;
  font-family: 'Lucida Sans', Arial, sans-serif;
  font-size: 16px;
  line-height: 26px;
  margin: 0;
}

li a:hover {
  background-color: #34C6CD;
}

.vertical-menu {
  width: auto;
}

.vertical-menu a {
  background-color: #000;
  color: white;
  display: block;
  padding: 12px;
  text-decoration: none;
  text-align: center;
  vertical-align: middle;
}

.vertical-menu a:hover {
  background-color: #34C6CD;
}

.active {
  background-color: #A66F00;
  color: white;
}

.info {
  margin: 10px 0px;
  padding:12px;
  color: white;
  background-color: #333;
}

.container {
  overflow-x: auto;
  white-space: nowrap;
}

table {
  border-collapse: collapse;
  width: 100%;
}

th, td {
  text-align: left;
  padding: 8px;
}

tr:nth-child(even){
  background-color: #f2f2f2
}

#github {
  height: 1em;
}

`)
