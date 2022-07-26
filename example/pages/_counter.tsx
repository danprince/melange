import "./_counter.css";
import { FunctionComponent, h } from "preact";
import { useState } from "preact/hooks";

let Counter: FunctionComponent = () => {
  let [count, setCount] = useState(0);
  return <button onClick={() => setCount(count + 1)}>{count}</button>
}

export default Counter;
