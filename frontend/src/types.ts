export type DataRow = {
  Timestamp: number;
  Data: number[];
};

interface ChartOptions {
  Title: string;
  XLabel: string;
  YLabel: string;
  YMin: number;
  YMax: number;
}

export interface Metadata {
  WindowSize: number;
  Columns: string[];
  YUnit: string;
  ChartOptions: ChartOptions;
}