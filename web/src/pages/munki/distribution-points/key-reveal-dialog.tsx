import { Copy, Eye, EyeOff } from "lucide-react";
import { type ReactNode, useState } from "react";
import { toast } from "sonner";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  InputGroup,
  InputGroupAddon,
  InputGroupButton,
  InputGroupInput,
} from "@/components/ui/input-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
const KEY_MASK = "••••••••••••••••••••••••";
export function KeyRevealDialog({
  title,
  description,
  value,
  open,
  onOpenChange,
}: {
  title: string;
  description: string;
  value: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const [visible, setVisible] = useState(false);
  async function copyKey() {
    try {
      await navigator.clipboard.writeText(value);
      toast.success("Key copied.");
    } catch {
      toast.error("Could not copy to clipboard.");
    }
  }
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-2xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        <InputGroup>
          <InputGroupInput
            readOnly
            className="font-mono"
            value={visible ? value : KEY_MASK}
            title={visible ? value : undefined}
          />
          <InputGroupAddon align="inline-end">
            <KeyAction label="Copy Key" onClick={() => void copyKey()}>
              <Copy />
            </KeyAction>
            <KeyAction
              label={visible ? "Hide Key" : "Show Key"}
              onClick={() => setVisible((current) => !current)}
            >
              {visible ? <EyeOff /> : <Eye />}
            </KeyAction>
          </InputGroupAddon>
        </InputGroup>

        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
function KeyAction({
  label,
  onClick,
  children,
}: {
  label: string;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger render={<InputGroupButton size="icon-sm" onClick={onClick} />}>
        {children}
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}
