(defun local/split ()
  (interactive)
  (save-excursion
    (goto-char (point-min))
    (while (re-search-forward "^سورة \\(.*\\)\n\n" nil t 1)
      (let ((name (concat (match-string 1) ".txt"))
            (start (match-end 0))
            (end (save-excursion
                   (if (re-search-forward "^سورة \\(.*\\)\n" nil t 1)
                       (1- (match-beginning 0))
                     (point-max)))))
        (write-region start end name)
        (with-temp-buffer
          (insert-file-contents name)
          (delete-matching-lines "بسم الله الرحمن الرحيم\n")
          (delete-matching-lines "^\s*$")
          (while (re-search-forward "\\(([0-9]*)\\) " nil t 1)
            (replace-match "\\1\n"))
          (write-region (point-min) (point-max) name))))))
