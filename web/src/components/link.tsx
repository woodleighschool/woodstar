import { createLink } from "@tanstack/react-router";
import { forwardRef, type ComponentPropsWithoutRef } from "react";

const LinkAnchor = forwardRef<HTMLAnchorElement, ComponentPropsWithoutRef<"a">>(function LinkAnchor(
  { children, ...props },
  ref,
) {
  return (
    <a ref={ref} data-link {...props}>
      {children}
    </a>
  );
});

const Link = createLink(LinkAnchor);

export { Link };
