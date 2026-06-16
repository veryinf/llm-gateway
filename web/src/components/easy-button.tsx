import { useState } from 'react';
import { Button } from './ui/button';
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from './ui/dropdown-menu';
import { Tooltip, TooltipContent, TooltipTrigger } from './ui/tooltip';
import { omit } from 'lodash-es';
import { Check, Copy } from 'lucide-react';

type EasyButtonProps = React.ComponentProps<typeof Button> & {
  tooltip?: string | React.ReactNode;
  tooltipProps?: Pick<React.ComponentProps<typeof TooltipContent>, 'side'>;
  options?: { label: string | React.ReactNode; value: string | number; onClick?: (key: string | number) => void; }[];
};

export function EasyButton(props: EasyButtonProps) {
  let output = <Button {...omit(props, ['tooltip', 'options', 'tooltipProps'])} />;
  if (props.options) {
    output = <DropdownMenuTrigger asChild>{output}</DropdownMenuTrigger>;
  }
  if (props.tooltip) {
    output = (
      <Tooltip>
        <TooltipTrigger asChild>{output}</TooltipTrigger>
        <TooltipContent {...props.tooltipProps}>
          <p>{props.tooltip}</p>
        </TooltipContent>
      </Tooltip>
    );
  }
  if (props.options) {
    output = (
      <DropdownMenu>
        {output}
        <DropdownMenuContent sideOffset={14}>
          {props.options.map((option) => (
            <DropdownMenuItem key={option.value} defaultValue={option.value} onClick={() => option.onClick && option.onClick(option.value)}>
              {option.label}
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    );
  }
  return output;
}


export function CopyButton({ text }: { text: string; }) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    await navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <Button variant="ghost" size="icon" className="h-6 w-6 shrink-0" onClick={handleCopy}>
      {copied ? <Check className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
    </Button>
  );
}