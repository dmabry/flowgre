// Use of this source code is governed by Apache License 2.0
// that can be found in the LICENSE file.

// Web is used to provide status

package templates

const DashboardTpl = `
<!DOCTYPE html>
<html>
<head>
<title>{{.Title}}</title>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<link rel="stylesheet" href="https://www.w3schools.com/w3css/4/w3.css">
<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Raleway">
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/4.7.0/css/font-awesome.min.css">
<script
  src="https://code.jquery.com/jquery-3.6.1.slim.min.js"
  integrity="sha256-w8CvhFs7iHNVUtnSP0YKEg00p9Ih13rlL9zGqvLdePA="
  crossorigin="anonymous">
</script>
<style>
html,body,h1,h2,h3,h4,h5 {font-family: "Raleway", sans-serif}
</style>
</head>
<body class="w3-light-grey">

<!-- Top container -->
<div class="w3-bar w3-top w3-black w3-large" style="z-index:4">
  <button class="w3-bar-item w3-button w3-hide-large w3-hover-none w3-hover-text-light-grey" onclick="w3_open();"><i class="fa fa-bars"></i>  Menu</button>
  <span class="w3-bar-item w3-right">Flowgre</span>
</div>

<!-- Overlay effect when opening sidebar on small screens -->
<div class="w3-overlay w3-hide-large w3-animate-opacity" onclick="w3_close()" style="cursor:pointer" title="close side menu" id="myOverlay"></div>

<!-- !PAGE CONTENT! -->
<div class="w3-main" style="margin-top:43px;">

  <!-- Header -->
  <header class="w3-container" style="padding-top:22px">
    <h5><b><i class="fa fa-dashboard"></i> Dashboard</b></h5>
  </header>
  <div class="w3-row-padding w3-margin-bottom">
    <div class="w3-quarter">
      <div class="w3-container w3-red w3-padding-16">
        <div class="w3-left"><i class="fa fa-users w3-xxxlarge"></i></div>
        <div class="w3-right">
          <h3>{{.ConfigOut.Workers}}</h3>
        </div>
        <div class="w3-clear"></div>
        <h4>Workers</h4>
      </div>
    </div>
    <div class="w3-quarter">
      <div class="w3-container w3-blue w3-padding-16">
        <div class="w3-left"><i class="fa fa-share-alt w3-xxxlarge"></i></div>
        <div class="w3-right">
          <h3>{{.StatsTotal.FlowsSent}}</h3>
        </div>
        <div class="w3-clear"></div>
        <h4>Flows</h4>
      </div>
    </div>
    <div class="w3-quarter">
      <div class="w3-container w3-teal w3-padding-16">
        <div class="w3-left"><i class="fa fa-circle w3-xxxlarge"></i></div>
        <div class="w3-right">
          <h3>{{.StatsTotal.Cycles}}</h3>
        </div>
        <div class="w3-clear"></div>
        <h4>Cycles</h4>
      </div>
    </div>
    <div class="w3-quarter">
      <div class="w3-container w3-orange w3-text-white w3-padding-16">
        <div class="w3-left"><i class="fa fa-cloud-download w3-xxxlarge"></i></div>
        <div class="w3-right">
          <h3>{{.StatsTotal.BytesSent}}</h3>
        </div>
        <div class="w3-clear"></div>
        <h4>BytesSent</h4>
      </div>
    </div>
  </div>

  <div class="w3-panel">
    <div class="w3-row-padding" style="margin:0 -16px">
      <div>
        <h5>Worker Details</h5>
        <table class="w3-table w3-striped w3-white">
          <th></th>
          <th>Worker ID</th>
          <th>Source ID</th>
          <th>Flows Sent</th>
          <th>Cycles</th>
          <th>Bytes Sent</th>
          {{ range $worker, $value := .StatsMapOut }}
          <tr>
            <td><i class="fa fa-user w3-text-blue w3-large"></i></td>
            <td>{{ $worker }}</td>
            <td><i>{{ $value.SourceID }}</i></td>
            <td><i>{{ $value.FlowsSent }}</i></td>
            <td><i>{{ $value.Cycles }}</i></td>
            <td><i>{{ $value.BytesSent }}</i></td>
          </tr>
          {{else}}
          <tr>
            <td><i class="fa fa-user w3-text-blue w3-large"></i></td>
            <td>No Worker Stats</td>
            <td></td>
            <td></td>
            <td></td>
            <td></td>
          </tr>
          {{end}}
        </table>
      </div>
    </div>
  </div>
  <div class="w3-panel">
    <div class="w3-row-padding" style="margin:0 -16px;width:300px">
      <div>
        <h5>Config Details</h5>
        <table class="w3-table w3-striped w3-white">
          <th>Item</th>
          <th>Value</th>
            <tr>
              <td>Target Server</td>
              <td>{{.ConfigOut.Server}}</td>
            </tr>
            <tr>
              <td>Target Port</td>
              <td>{{.ConfigOut.DstPort}}</td>
            </tr>
            <tr>
              <td>Delay</td>
              <td>{{.ConfigOut.Delay}} ms</td>
            </tr>
            <tr>
              <td>Number of Workers</td>
              <td>{{.ConfigOut.Workers}}</td>
            </tr>
        </table>
      </div>
    </div>
  </div>
  <hr>
  <!-- Footer -->
  <footer class="w3-container w3-padding-16 w3-light-grey">
    <h4>Flowgre - Refresh in <span id='timer'>30 secs</span></h4>
    <p>Powered by <a href="https://www.w3schools.com/w3css/default.asp" target="_blank">w3.css</a></p>
  </footer>

  <!-- End page content -->
</div>

<script>
// Get the Sidebar
var mySidebar = document.getElementById("mySidebar");

// Get the DIV with overlay effect
var overlayBg = document.getElementById("myOverlay");

// Toggle between showing and hiding the sidebar, and add overlay effect
function w3_open() {
  if (mySidebar.style.display === 'block') {
    mySidebar.style.display = 'none';
    overlayBg.style.display = "none";
  } else {
    mySidebar.style.display = 'block';
    overlayBg.style.display = "block";
  }
}

// Close the sidebar with the close button
function w3_close() {
  mySidebar.style.display = "none";
  overlayBg.style.display = "none";
}

$(document).ready(function(){

  var count=30;
  var counter=setInterval(timer, 1000);

  function timer(){
    count=count-1;
      if (count <= 0){
        clearInterval(counter);
        location.reload()
        return;
      }
    document.getElementById("timer").innerHTML=count + " secs"; // watch for spelling
  }    
});
</script>
</body>
</html>
`
