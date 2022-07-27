import "./_counter.css";
import { FunctionComponent, h } from "preact";
import { useState } from "preact/hooks";
import duneSrc from "./dune.png";

let Counter: FunctionComponent<{ count?: number }> = ({ count: initialCount = 0 }) => {
  let [count, setCount] = useState(initialCount);
  return (
    <div>
      <img src={duneSrc} width={300} />
      <br />
      <button onClick={() => setCount(count + 1)}>{count}</button>
    </div>
  );
}

export default Counter;
