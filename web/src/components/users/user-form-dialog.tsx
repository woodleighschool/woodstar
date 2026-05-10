import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useCreateUser, useUpdateUser, type User, type UserCreateBody, type UserUpdateBody } from "@/hooks/use-users";

type Role = User["role"];

interface BaseProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  // canChangeRole gates the role select. False when editing self.
  canChangeRole?: boolean;
}

interface CreateProps extends BaseProps {
  mode: "create";
}

interface EditProps extends BaseProps {
  mode: "edit";
  user: User;
}

export type UserFormDialogProps = CreateProps | EditProps;

export function UserFormDialog(props: UserFormDialogProps) {
  // Body lives inside DialogContent and remounts on each open, so its state
  // resets without an effect. Key on user.id covers the case of switching
  // between edit targets without closing.
  const bodyKey = props.mode === "create" ? "create" : `edit-${props.user.id}`;

  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent className="sm:max-w-md">
        {props.open ? (
          <UserFormBody
            key={bodyKey}
            mode={props.mode}
            editing={props.mode === "edit" ? props.user : null}
            canChangeRole={props.canChangeRole ?? true}
            onClose={() => props.onOpenChange(false)}
          />
        ) : null}
      </DialogContent>
    </Dialog>
  );
}

interface UserFormBodyProps {
  mode: "create" | "edit";
  editing: User | null;
  canChangeRole: boolean;
  onClose: () => void;
}

function UserFormBody({ mode, editing, canChangeRole, onClose }: UserFormBodyProps) {
  const create = useCreateUser();
  const update = useUpdateUser();
  const pending = create.isPending || update.isPending;
  const submitError = mode === "create" ? create.error : update.error;

  const [email, setEmail] = useState(editing?.email ?? "");
  const [name, setName] = useState(editing?.name ?? "");
  const [role, setRole] = useState<Role>(editing?.role ?? "viewer");
  const [password, setPassword] = useState("");

  async function handleSubmit() {
    if (mode === "create") {
      const body: UserCreateBody = { email, name, role, password };
      await create.mutateAsync(body);
      onClose();
      return;
    }

    const body: UserUpdateBody = {
      name,
      role: canChangeRole ? role : editing!.role,
    };
    if (password.trim() !== "") body.password = password;
    await update.mutateAsync({ id: editing!.id, body });
    onClose();
  }

  const title = mode === "create" ? "Add user" : "Edit user";
  const description =
    mode === "create"
      ? "Create a new Woodstar user. Roles control whether the user can manage other users and secrets."
      : "Update name, role, or reset password. Email cannot change.";

  return (
    <>
      <DialogHeader>
        <DialogTitle>{title}</DialogTitle>
        <DialogDescription>{description}</DialogDescription>
      </DialogHeader>

      <form
        className="grid gap-3"
        onSubmit={(event) => {
          event.preventDefault();
          void handleSubmit();
        }}
      >
        <div className="grid gap-1.5">
          <Label htmlFor="user-email">Email</Label>
          <Input
            id="user-email"
            type="email"
            required
            autoComplete="off"
            value={email}
            disabled={mode === "edit"}
            onChange={(event) => setEmail(event.target.value)}
          />
        </div>

        <div className="grid gap-1.5">
          <Label htmlFor="user-name">Name</Label>
          <Input
            id="user-name"
            type="text"
            autoComplete="off"
            value={name}
            onChange={(event) => setName(event.target.value)}
            placeholder={mode === "create" ? "Optional, defaults to email" : ""}
          />
        </div>

        <div className="grid gap-1.5">
          <Label htmlFor="user-role">Role</Label>
          <Select value={role} onValueChange={(value) => setRole(value as Role)} disabled={!canChangeRole}>
            <SelectTrigger id="user-role" className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="admin">admin</SelectItem>
              <SelectItem value="viewer">viewer</SelectItem>
            </SelectContent>
          </Select>
          {!canChangeRole ? <p className="text-xs text-muted-foreground">You cannot change your own role.</p> : null}
        </div>

        <div className="grid gap-1.5">
          <Label htmlFor="user-password">Password{mode === "edit" ? " (leave blank to keep current)" : ""}</Label>
          <Input
            id="user-password"
            type="password"
            autoComplete="new-password"
            required={mode === "create"}
            minLength={mode === "create" ? 12 : undefined}
            value={password}
            onChange={(event) => setPassword(event.target.value)}
            placeholder={mode === "create" ? "Min 12 characters" : ""}
          />
        </div>

        {submitError ? <p className="text-sm text-destructive">{submitError.message}</p> : null}

        <DialogFooter className="pt-2">
          <DialogClose asChild>
            <Button type="button" variant="ghost" size="sm" disabled={pending}>
              Cancel
            </Button>
          </DialogClose>
          <Button type="submit" size="sm" disabled={pending}>
            {mode === "create" ? "Create user" : "Save changes"}
          </Button>
        </DialogFooter>
      </form>
    </>
  );
}
