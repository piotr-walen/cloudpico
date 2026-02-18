// Type declarations for uPlot library (loaded globally)
declare namespace uPlotTypes {
  interface Options {
    title?: string;
    width?: number;
    height?: number;
    scales?: Record<string, Scale>;
    series?: Series[];
    axes?: Axis[];
    legend?: Legend;
    cursor?: Cursor;
  }

  interface Scale {
    time?: boolean;
    range?: [number, number] | ((self: UPlotInstance, initMin: number, initMax: number) => [number, number]);
    min?: number;
    max?: number;
    label?: string;
    side?: number;
  }

  interface Series {
    label?: string;
    stroke?: string;
    width?: number;
    scale?: string;
  }

  interface Axis {
    stroke?: string;
    grid?: { show?: boolean; stroke?: string; width?: number };
    time?: boolean;
    scale?: string;
    show?: boolean;
    label?: string;
    values?: (self: UPlotInstance, splits: number[]) => string[];
  }

  interface Legend {
    live?: boolean;
  }

  interface Cursor {
    show?: boolean;
    x?: boolean;
    y?: boolean;
  }

  type AlignedData = (number | null)[][];

  interface UPlotInstance {
    destroy(): void;
    setSize(size: { width: number; height: number }): void;
    scales: Record<string, Scale>;
  }

  class UPlot {
    constructor(opts: Options, data: AlignedData, container: HTMLElement);
    destroy(): void;
    setSize(size: { width: number; height: number }): void;
    scales: Record<string, Scale>;
  }
}

declare const uPlot: typeof uPlotTypes.UPlot;
type uPlot = uPlotTypes.UPlot;
