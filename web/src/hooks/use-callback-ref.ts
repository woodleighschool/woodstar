import * as React from "react";

/**
 * A custom hook that converts a callback to a ref to avoid triggering re-renders when passed as a
 * prop or avoid re-executing effects when passed as a dependency
 */
function useCallbackRef<T extends (...args: never[]) => unknown>(callback: T | undefined): T {
  const callbackRef = React.useRef(callback);

  React.useLayoutEffect(() => {
    callbackRef.current = callback;
  }, [callback]);

  return React.useCallback(((...args) => callbackRef.current?.(...args)) as T, []);
}

export { useCallbackRef };
