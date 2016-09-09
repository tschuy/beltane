var getDumpsByDay = function(date) {
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
  }

  function processRequest() {
    if (httpRequest.readyState === XMLHttpRequest.DONE) {
      if (httpRequest.status === 200) {
        for (var dump of JSON.parse(httpRequest.responseText)) {
          dump.created_at = new Date(dump.created_at)
          appendElement(dump);
        }
      } else {
        alert('request failed');
      }
    }
  }
  if (date !== undefined) {
    makeRequest(`/v1/dumps?date=${date.getFullYear()}/${padZero(date.getMonth()+1)}/${padZero(date.getDate())}`);
  } else {
    makeRequest('/v1/dumps');
  }
}

var changeDate = function() {
  // we need to be sure that the .getDate()/.getX() functions won't accidentally return wrong day
  // during timezone conversion
  var selected_date = document.getElementsByName("date_box")[0].value;
  var temp_time = new Date(selected_date + " GMT");
  var local_time = new Date(temp_time.getTime() + temp_time.getTimezoneOffset()*60000);
  getDumpsByDay(local_time);
}

var padZero = function(month) {
  // .getMonth() returns an int of the month -- pad if necessary
  return ('0'+(month)).slice(-2);
}

var monthNames = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];

var appendElement = function(dump) {
  console.log(dump)
  var base_html = `
  <tr class="row">
    <td class="element">${padZero(dump.created_at.getDate())} ${monthNames[dump.created_at.getMonth()]} ${dump.created_at.getFullYear()} ${padZero(dump.created_at.getHours() + 1)}:${dump.created_at.getMinutes()}</td>
    <td class="element"><input type="text" value="${dump.machine_id}" class="input_data"/></td>
    <td class="element"><input type="text" value="${dump.sha.substring(0,8)}" title="${dump.sha}" class="input_data" /></td>
    <td class="element"><a class="download" href="${dump.download_url}">Download</a></td>
  </tr>`
  var tbody = document.getElementById('dump_list');
  tbody.insertAdjacentHTML('beforeend', base_html);

}

window.onload = function() {
  getDumpsByDay();
}
