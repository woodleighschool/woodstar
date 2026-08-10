import { createLink } from "@tanstack/react-router";
import { forwardRef, type ComponentPropsWithoutRef } from "react";

const LinkAnchor = forwardRef<HTMLAnchorElement, ComponentPropsWithoutRef<"a">>(function LinkAnchor(
  { children, ...props },
  ref,
) {
  return (
    <a ref={ref} {...props}>
      {children}
    </a>
  );
});

const TextLinkAnchor = forwardRef<HTMLAnchorElement, ComponentPropsWithoutRef<"a">>(
  function TextLinkAnchor({ children, ...props }, ref) {
    return (
      <a ref={ref} data-text-link {...props}>
        {children}
      </a>
    );
  },
);

const Link = createLink(LinkAnchor);
const TextLink = createLink(TextLinkAnchor);

export { Link, TextLink };
