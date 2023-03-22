import * as echarts from "echarts";

type DataRow = {
  Timestamp: number;
  Data: number[];
};

interface DataItem {
  value: [number, number];
}

function main() {
  const dom = document.getElementById("container")!;

  const myChart = echarts.init(dom, undefined, {
    renderer: "canvas",
    useDirtyRect: false,
  });

  let init_x: number;
  const series_data: DataItem[] = [];

  const options: echarts.EChartsOption = {
    xAxis: {
      type: "value",
    },
    yAxis: {
      type: "value",
    },
    series: [
      {
        type: "line",
        data: series_data,
      },
    ],
  };

  const hostname = `ws://${location.hostname}:8080/ws`;
  console.log(`connecting to ${hostname}`);
  const socket = new WebSocket(hostname);

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
    const data: DataRow = JSON.parse(event.data);
    if (init_x == undefined) {
      init_x = data.Timestamp;
    }

    series_data.push({
      value: [data.Timestamp - init_x, data.Data[0]],
    });

    myChart.setOption({
      series: [
        {
          data: series_data,
        },
      ],
    });
  });

  myChart.setOption(options);
  window.addEventListener("resize", () => {
    myChart.resize();
  });
}

window.addEventListener("load", main);
