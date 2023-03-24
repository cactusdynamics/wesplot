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
  WindowSize: number;
  Columns: string[];
  YUnit: string;
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
  const data_arrays: DataItem[][] = [];
  for (const _ of metadata.Columns) {
    data_arrays.push([]);
  }
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
        formatter: (value: String) => `${value} ${metadata.YUnit}`,
      },
    },
    textStyle: {
      fontSize: 16,
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
          type: ["line", "bar"],
        },
      },
    },
  };
  const additional_options: echarts.EChartsOption = {
    toolbox: {
      feature: {
        myToggleTooltip: {
          show: true,
          title: "Toggle Tooltip",
          icon: "path://M432.45,595.444c0,2.177-4.661,6.82-11.305,6.82c-6.475,0-11.306-4.567-11.306-6.82s4.852-6.812,11.306-6.812C427.841,588.632,432.452,593.191,432.45,595.444L432.45,595.444z M421.155,589.876c-3.009,0-5.448,2.495-5.448,5.572s2.439,5.572,5.448,5.572c3.01,0,5.449-2.495,5.449-5.572C426.604,592.371,424.165,589.876,421.155,589.876L421.155,589.876z M421.146,591.891c-1.916,0-3.47,1.589-3.47,3.549c0,1.959,1.554,3.548,3.47,3.548s3.469-1.589,3.469-3.548C424.614,593.479,423.062,591.891,421.146,591.891L421.146,591.891zM421.146,591.891",
          onclick: function () {
            if (
              (options.tooltip! as echarts.TooltipComponentOption).trigger ===
              "none"
            ) {
              options.tooltip = {
                trigger: "axis",
                axisPointer: {
                  type: "cross",
                },
              };
            } else {
              options.tooltip = {
                trigger: "none",
              };
            }
            myChart.setOption(options);
          },
        },
      },
    },
  };
  merge(options, additional_options);
  merge(options, metadata.EChartsOption);
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
    const series: echarts.SeriesOption[] = [];

    // TODO: inefficient, but OK for now.
    for (const row of rows) {
      if (init_x == undefined) {
        init_x = row.Timestamp;
      }
      for (const [i, _] of metadata.Columns.entries()) {
        data_arrays[i].push({
          value: [row.Timestamp - init_x, row.Data[i]],
        });
        if (data_arrays[i].length > metadata.WindowSize) {
          data_arrays[i].shift();
        }
      }
    }

    for (const [i, column] of metadata.Columns.entries()) {
      series.push({
        type: "line",
        name: column,
        data: data_arrays[i],
      });
    }

    myChart.setOption({
      series: series,
    });
  });

  myChart.setOption(options);
  window.addEventListener("resize", () => {
    myChart.resize();
  });
}

window.addEventListener("load", main);
