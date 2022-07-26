import "./_counter.css";
import { FunctionComponent, h } from "preact";
import { useState } from "preact/hooks";

let Counter: FunctionComponent<{ count?: number }> = ({ count: initialCount = 0 }) => {
  let [count, setCount] = useState(initialCount);
  return <button onClick={() => setCount(count + 1)}>{count}</button>
}

export default Counter;
