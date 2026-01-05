export interface WesplotOptions {
  Title: string;
  Columns: string[];
  XLabel: string;
  YLabel: string;
  YMin?: number;
  YMax?: number;
  XMin?: number;
  XMax?: number;
  YUnit: string;
  ChartType: string;
}

export interface Metadata {
  WindowSize: number;
  XIsTimestamp: boolean;
  RelativeStart: boolean;
  WesplotOptions: WesplotOptions;
}
