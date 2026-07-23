import { revalidateLogic, useForm } from "@tanstack/react-form";
import { z } from "zod";

import { FormField } from "@/components/form-field";
import { PendingButton } from "@/components/pending-button";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { FieldGroup } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useCreateUser } from "@/hooks/use-users";
import { emailAddress } from "@/lib/form-validation";
import { USER_ROLE_OPTIONS, USER_ROLE_VALUES, type UserRole } from "@/lib/users";
import { isOneOf } from "@/lib/utils";
interface Props {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}
export function UserFormDialog({ open, onOpenChange }: Props) {
  // Body remounts on each open, so its form state resets without an effect.
  return open ? <UserFormBody onClose={() => onOpenChange(false)} /> : null;
}
function UserFormBody({ onClose }: { onClose: () => void }) {
  const create = useCreateUser();
  const form = useForm({
    defaultValues: {
      email: "",
      name: "",
      role: "viewer" as UserRole,
      password: "",
    },
    validationLogic: revalidateLogic({
      mode: "submit",
      modeAfterSubmission: "change",
    }),
    validators: {
      onDynamic: z.object({
        email: emailAddress(),
        name: z.string().trim(),
        role: z.enum(["admin", "viewer"]),
        password: z.string().min(12, "Password must be at least 12 characters."),
      }),
    },
    onSubmit: async ({ value }) => {
      await create.mutateAsync({
        email: value.email.trim(),
        name: value.name.trim(),
        role: value.role,
        password: value.password,
      });
      onClose();
    },
  });
  function requestClose() {
    if (create.isPending || form.state.isSubmitting) return;
    onClose();
  }
  return (
    <Dialog
      open
      onOpenChange={(nextOpen) => {
        if (!nextOpen) requestClose();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Create User</DialogTitle>
          <DialogDescription>
            Roles control whether the user can manage other users and enrollments.
          </DialogDescription>
        </DialogHeader>

        <form
          noValidate
          className="flex flex-col gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            event.stopPropagation();
            void form.handleSubmit();
          }}
        >
          <FieldGroup className="gap-4">
            <form.Field name="email">
              {(field) => (
                <FormField field={field} label="Email" htmlFor="user-email" required>
                  {(control) => (
                    <Input
                      {...control}
                      type="email"
                      required
                      autoComplete="off"
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </FormField>
              )}
            </form.Field>

            <form.Field name="name">
              {(field) => (
                <FormField field={field} label="Name" htmlFor="user-name">
                  {(control) => (
                    <Input
                      {...control}
                      type="text"
                      autoComplete="off"
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </FormField>
              )}
            </form.Field>

            <form.Field name="role">
              {(field) => (
                <FormField field={field} label="Role" htmlFor="user-role">
                  {(control) => (
                    <Select
                      items={USER_ROLE_OPTIONS}
                      value={field.state.value}
                      onValueChange={(value) => {
                        if (isOneOf(value, USER_ROLE_VALUES)) field.handleChange(value);
                      }}
                    >
                      <SelectTrigger {...control} className="w-full">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectGroup>
                          {USER_ROLE_OPTIONS.map((option) => (
                            <SelectItem key={option.value} value={option.value}>
                              {option.label}
                            </SelectItem>
                          ))}
                        </SelectGroup>
                      </SelectContent>
                    </Select>
                  )}
                </FormField>
              )}
            </form.Field>

            <form.Field name="password">
              {(field) => (
                <FormField field={field} label="Password" htmlFor="user-password" required>
                  {(control) => (
                    <Input
                      {...control}
                      type="password"
                      autoComplete="new-password"
                      required
                      minLength={12}
                      placeholder="Min 12 characters"
                      value={field.state.value}
                      onBlur={field.handleBlur}
                      onChange={(event) => field.handleChange(event.target.value)}
                    />
                  )}
                </FormField>
              )}
            </form.Field>
          </FieldGroup>

          <DialogFooter className="pt-2">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={create.isPending}
              onClick={requestClose}
            >
              Cancel
            </Button>
            <form.Subscribe selector={(state) => state.isSubmitting}>
              {(isSubmitting) => (
                <PendingButton isPending={isSubmitting} type="submit" size="sm">
                  Create
                </PendingButton>
              )}
            </form.Subscribe>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
