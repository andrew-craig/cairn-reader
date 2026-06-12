import type { Article } from '@cairn/shared';
import './ArticleRow.css';

interface ArticleRowProps {
  article: Article;
  onSelect: (article: Article) => void;
}

// A single reading-list entry: title, source/author, excerpt and an optional
// lead image. Mirrors mobile's ArticleRow (muted once read) while adding the
// excerpt the web list calls for.
export default function ArticleRow({ article, onSelect }: ArticleRowProps) {
  return (
    <li className="article-row">
      <button
        type="button"
        className={`article-row__button${article.isRead ? ' article-row__button--read' : ''}`}
        onClick={() => onSelect(article)}
      >
        <div className="article-row__text">
          <h2 className="article-row__title">{article.title}</h2>
          <p className="article-row__meta">{article.author || 'Unknown'}</p>
          {article.description && (
            <p className="article-row__excerpt">{article.description}</p>
          )}
        </div>
        {article.imageUrl && (
          <div className="article-row__image-frame">
            <img
              className="article-row__image"
              src={article.imageUrl}
              alt=""
              loading="lazy"
            />
          </div>
        )}
      </button>
    </li>
  );
}
