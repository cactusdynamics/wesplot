import * as echarts from "echarts";
import "./app.css";
import "@fortawesome/fontawesome-free/css/fontawesome.css";
import "@fortawesome/fontawesome-free/css/solid.css";

type DataRow = {
  Timestamp: number;
  Data: number[];
};

interface DataItem {
  value: [number, number];
}

interface Metadata {
  RollingWindowSize: number;
  EChartsOption: echarts.EChartsOption;
}

async function main() {
  const response = await fetch(`http://${location.hostname}:8080/metadata`);
  const metadata = await response.json();
  const dom = document.getElementById("plot")!;

  const myChart = echarts.init(dom, undefined, {
    renderer: "canvas",
    useDirtyRect: false,
  });

  let init_x: number;
  const series_data: DataItem[] = [];

  const options: echarts.EChartsOption = {
    ...metadata.EChartsOption,
    xAxis: {
      type: "time",
    },
    yAxis: {
      type: "value",
    },
    grid: {
      left: 30,
      right: 30,
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
    if (series_data.length > metadata.RollingWindowSize) {
      series_data.shift();
    }

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
