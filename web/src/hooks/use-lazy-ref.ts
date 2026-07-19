import * as React from "react";

function useLazyRef<T>(fn: () => T): { current: T } {
  const [ref] = React.useState(() => ({ current: fn() }));
  return ref;
}

export { useLazyRef };
