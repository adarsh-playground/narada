type WordMeaningsProps = {
  text: string;
  className?: string;
};

export function WordMeanings({ text, className = '' }: WordMeaningsProps) {
  const meanings = text
    .split(';')
    .map((meaning) => meaning.trim())
    .filter(Boolean);

  return (
    <div className={`word-meanings ${className}`.trim()}>
      {meanings.map((meaning, index) => (
        <span key={`${meaning}-${index}`}>{meaning}</span>
      ))}
    </div>
  );
}
