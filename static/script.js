var getDumpsByDay = function(date, num) {
  // date of request (Date object)
  // number of days to display starting from that date
  var httpRequest;
  function makeRequest(url) {
    httpRequest = new XMLHttpRequest();

    if (!httpRequest) {
      alert('request failed');
      return false;
    }
    httpRequest.onreadystatechange = processRequest;
    httpRequest.open('GET', url);
    httpRequest.send();

    var tbody = document.getElementById('dump_list');
    tbody.innerHTML = "<h2 class='centered'>Loading...</h2>";

  }

  function processRequest() {
    var tbody = document.getElementById('dump_list');
    if (httpRequest.readyState === XMLHttpRequest.DONE) {
      if (httpRequest.status === 200) {
        tbody.innerHTML = "";
        for (var dump of JSON.parse(httpRequest.responseText)) {
          temp_date = new Date(dump.created_at)
          dump.created_at = new Date(temp_date.getTime() + temp_date.getTimezoneOffset() * 60000);
          appendElement(dump);
        }
      } else {
        tbody.innerHTML = "<h2 class='centered'>Could not load dumps</h2>";
      }
    }
  }
  if (num === undefined) {
    num = 20;
  }
  if (date !== undefined) {
    date = new Date(date.getTime() + date.getTimezoneOffset()*60000);
    makeRequest(`/v1/dumps?date=${date.getFullYear()}/${padZero(date.getMonth()+1)}/${padZero(date.getDate())}&num=${num}`);
  } else {
    makeRequest('/v1/dumps?num=${num}');
  }
}

var getParam = function(variable) {
  var query = window.location.search.substring(1);
  var vars = query.split("&");
  for (var i=0; i<vars.length; i++) {
    var pair = vars[i].split("=");
    if (pair[0] == variable) {
      return pair[1];
    }
  }
  return(false);
}

var changeDate = function() {
  // we need to be sure that the .getDate()/.getX() functions won't accidentally return wrong day
  // during timezone conversion
  var selected_date = document.getElementsByName("date_box")[0].value;
  if (selected_date === "") {
    return;
  }
  var count = document.getElementsByName("num")[0].value;
  var temp_time = new Date(selected_date + " GMT");
  var local_time = new Date(temp_time.getTime() + temp_time.getTimezoneOffset()*60000);
  return getDumpsByDay(local_time, count);
}

var padZero = function(month) {
  // .getMonth() returns an int of the month -- pad if necessary
  return ('0'+(month)).slice(-2);
}

var monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];

var fullSha = function(element) {
  element.innerHTML = element.getAttribute("title");
}

var appendElement = function(dump) {
  var sha = getParam('sha');
  var highlight="";
  if (sha && dump.sha.includes(sha)) {
    highlight = "highlight";
  }

  var base_html = `
  <tr class="row ${highlight}">
    <td class="element">${padZero(dump.created_at.getDate())} ${monthNames[dump.created_at.getMonth()]} ${dump.created_at.getFullYear()} ${padZero(dump.created_at.getHours())}:${padZero(dump.created_at.getMinutes())}</td>
    <td class="element"><span class="input_data">${dump.machine_id}</td>
    <td class="element"><span title="${dump.sha}" class="input_data" onclick="fullSha(this)" oncontextmenu="fullSha(this)">${dump.sha.substring(0,8)}</span></td>
    <td class="element"><a class="download" href="${dump.download_url}">Download</a></td>
  </tr>`
  var tbody = document.getElementById('dump_list');
  tbody.insertAdjacentHTML('beforeend', base_html);

}

window.onload = function() {
  // get date specified by url parameter, if specified
  var date = getParam('date');
  date = date ? new Date(date) : new Date();
  getDumpsByDay(date, 20);
}
