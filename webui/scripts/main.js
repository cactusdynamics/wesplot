let socket = new WebSocket(`ws://${location.host}/ws`);
var dom = document.getElementById('container');
var myChart = echarts.init(dom, null, {
  renderer: 'canvas',
  useDirtyRect: false
});

let init_x;
let series_data = [];
var option;
option = {
  xAxis: {
    type: 'value',
  },
  yAxis: {
    type: 'value',
  },
  series: [
    {
      type: 'line',
      data: series_data,
    }
  ]
};

window.addEventListener('resize', myChart.resize);

console.log("Attempting Connection...");

socket.addEventListener("open", () => {
  console.log("Successfully Connected");
});

socket.addEventListener("close", (event) => {
  console.log("Socket Closed Connection: ", event);
});

socket.addEventListener("error", (error) => {
  console.log("Socket Error: ", error);
});

socket.addEventListener("message", (event) => {
  const data = JSON.parse(event.data);
  if (init_x === undefined) {
    init_x = data.Timestamp
  }
  series_data.push({
    // name: now.toString(),
    value: [
      data.Timestamp - init_x,
      data.Data,
    ]
  });
  console.log(series_data)
  myChart.setOption({
    series: [
      {
        data: series_data
      }
    ]
  });
});

myChart.setOption(option);
