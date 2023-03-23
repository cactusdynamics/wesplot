import * as echarts from "echarts";
import "./app.css";
import "@fortawesome/fontawesome-free/css/fontawesome.css";
import "@fortawesome/fontawesome-free/css/solid.css";
import { merge } from "lodash";

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
  const metadata: Metadata = await response.json();
  const dom = document.getElementById("plot")!;

  const myChart = echarts.init(dom, undefined, {
    renderer: "canvas",
    useDirtyRect: false,
  });

  let init_x: number;
  const series_data: DataItem[] = [];

  const options: echarts.EChartsOption = {
    xAxis: {
      type: "value",
      nameLocation: "middle",
      nameGap: 35,
      nameTextStyle: {
        fontWeight: "bolder",
      },
      axisLabel: {
        fontSize: 16,
      },
    },
    yAxis: {
      type: "value",
      nameLocation: "middle",
      nameGap: 25,
      nameTextStyle: {
        fontWeight: "bolder",
      },
      axisLabel: {
        fontSize: 16,
        formatter: (value: String) => `${value} kg`,
      },
    },
    textStyle: {
      fontSize: 16,
    },
    grid: {
      left: 50,
      right: 50,
    },
    tooltip: {
      trigger: "axis",
      axisPointer: {
        type: "cross",
      },
    },
    title: {
      left: "center",
    },
    toolbox: {
      show: true,
      feature: {
        saveAsImage: {},
        dataView: {},
        dataZoom: {},
        magicType: {
          type: ['line', 'bar']
        },
      }
    },
    series: [
      {
        type: "line",
        data: series_data,
      },
    ],
  };
  merge(options, metadata.EChartsOption)
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
    const rows: DataRow[] = JSON.parse(event.data);

    // TODO: inefficient, but OK for now.
    for (const data of rows) {
      if (init_x == undefined) {
        init_x = data.Timestamp;
      }

      series_data.push({
        value: [data.Timestamp - init_x, data.Data[0]],
      });
      if (series_data.length > metadata.RollingWindowSize) {
        series_data.shift();
      }
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
