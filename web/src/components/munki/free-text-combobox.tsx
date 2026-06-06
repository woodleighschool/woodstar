import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";

export function FreeTextCombobox({
  id,
  name,
  value,
  options,
  placeholder,
  invalid,
  onBlur,
  onChange,
}: {
  id: string;
  name?: string;
  value: string;
  options: string[];
  placeholder?: string;
  invalid?: boolean;
  onBlur?: () => void;
  onChange: (value: string) => void;
}) {
  const selected = options.includes(value) ? value : null;

  return (
    <Combobox
      items={options}
      value={selected}
      inputValue={value}
      onInputValueChange={onChange}
      onValueChange={(next) => onChange(next ?? "")}
    >
      <ComboboxInput
        id={id}
        name={name}
        className="w-full"
        placeholder={placeholder}
        showClear={value !== ""}
        aria-invalid={invalid}
        onBlur={onBlur}
      />
      <ComboboxContent>
        <ComboboxEmpty>{options.length === 0 ? "No Values Available." : "No Values Found."}</ComboboxEmpty>
        <ComboboxList>
          {(option: string) => (
            <ComboboxItem key={option} value={option}>
              {option}
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
