import React from "react";
import { useEffect, useRef } from "react";
import { Grid } from "gridjs";
import "gridjs/dist/theme/mermaid.css";

export const GridHelloWorld: React.FC = () => {
  const wrapperRef = useRef(null);

  const grid = new Grid({
    columns: ['Name', 'Email', 'Phone Number'],
    data: [
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 2 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['John', 'john@example.com', '(353) 01 222 3333'],
      ['Mark', 'mark@gmail.com',   '(01) 22 888 4444']
    ],
    style: {
        td: {
          border: '5px solid #8e4bd5'
        },
        table: {
          'font-size': '15px'
        }
      }
  });

  useEffect(() => {
    if (wrapperRef.current) {
      grid.render(wrapperRef.current);
    }
  }, [grid, wrapperRef]);

  return <div ref={wrapperRef} />;
}