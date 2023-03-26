export type DataRow = {
  X: number;
  Ys: number[];
};

interface ChartOptions {
  Title: string;
  XLabel?: string;
  YLabel?: string;
  YMin?: number;
  YMax?: number;
}

export interface Metadata {
  WindowSize: number;
  Columns: string[];
  XIsTimestamp: boolean;
  YUnit: string;
  ChartOptions: ChartOptions;
}

export interface ChartButtons {
  zoom: HTMLButtonElement;
  resetzoom: HTMLButtonElement;
  pan: HTMLButtonElement;
  screenshot: HTMLButtonElement;
  settings: HTMLButtonElement;
}
