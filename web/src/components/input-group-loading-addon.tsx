import type { ComponentProps } from "react";

import { InputGroupAddon } from "@components/ui/input-group";
import { Spinner } from "@components/ui/spinner";

type InputGroupLoadingAddonProps = Omit<
  ComponentProps<typeof InputGroupAddon>,
  "align" | "children"
>;

export function InputGroupLoadingAddon(props: InputGroupLoadingAddonProps) {
  return (
    <InputGroupAddon align="inline-end" {...props}>
      <Spinner className="size-3.5" />
    </InputGroupAddon>
  );
}
