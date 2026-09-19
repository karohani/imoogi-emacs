;;; imoogi-target-scan.el --- Multi-target Org scan grouping -*- lexical-binding: t; -*-

;;; Commentary:

;; This layer preserves `imoogi-scan.el' as the single-file parser and
;; adds only target discovery/grouping for sync roots registered outside
;; the legacy single `imoogi-sync-root'.

;;; Code:

(require 'cl-lib)
(require 'subr-x)
(require 'imoogi-scan)

(defun imoogi-target-scan--directory-name (path)
  "Return PATH as an expanded directory name without resolving symlinks."
  (file-name-as-directory (expand-file-name path)))

(defun imoogi-target-scan--true-path (path)
  "Return PATH's canonical name when possible, otherwise its expansion."
  (condition-case nil
      (file-truename path)
    (file-error (expand-file-name path))))

(defun imoogi-target-scan--true-directory (path)
  "Return PATH's canonical directory name when possible."
  (file-name-as-directory (imoogi-target-scan--true-path path)))

(defun imoogi-target-scan--inside-directory-p (path directory)
  "Return non-nil when PATH is inside DIRECTORY after canonicalization."
  (let ((file (imoogi-target-scan--true-path path))
        (root (imoogi-target-scan--true-directory directory)))
    (string-prefix-p root file)))

(defun imoogi-target-scan--org-file-p (path)
  "Return non-nil when PATH names an Org file."
  (string-suffix-p ".org" path))

(defun imoogi-target-scan--covered-by-directory-p (file directories)
  "Return non-nil when FILE is already covered by one of DIRECTORIES."
  (cl-some (lambda (directory)
             (imoogi-target-scan--inside-directory-p file directory))
           directories))

(defun imoogi-target-scan--covering-group (groups file)
  "Return the first group in GROUPS whose root covers FILE."
  (cl-find-if (lambda (group)
                (imoogi-target-scan--inside-directory-p
                 file (plist-get group :root)))
              groups))

(defun imoogi-target-scan--make-group (root legacy)
  "Create an internal scan group for ROOT.
LEGACY is non-nil for the group created from the legacy sync root."
  (list :root (imoogi-target-scan--true-directory root)
        :legacy legacy
        :directory-targets nil
        :file-targets nil
        :failures nil
        :entries nil
        :census nil))

(defun imoogi-target-scan--find-group (groups root)
  "Return the scan group in GROUPS whose root is ROOT."
  (let ((canonical (imoogi-target-scan--true-directory root)))
    (cl-find canonical groups
             :key (lambda (group) (plist-get group :root))
             :test #'string=)))

(defun imoogi-target-scan--ensure-group (groups root legacy)
  "Ensure GROUPS has a group rooted at ROOT.
When LEGACY is non-nil, the existing group is marked legacy and moved
to the front by the caller."
  (let ((group (imoogi-target-scan--find-group groups root)))
    (if group
        (progn
          (when legacy (plist-put group :legacy t))
          groups)
      (append groups (list (imoogi-target-scan--make-group root legacy))))))

(defun imoogi-target-scan--add-to-group (groups root key value)
  "Append VALUE to KEY in the group rooted at ROOT inside GROUPS."
  (let ((group (imoogi-target-scan--find-group groups root)))
    (plist-put group key (append (plist-get group key) (list value))))
  groups)

(defun imoogi-target-scan--target-path (target)
  "Return TARGET's path."
  (plist-get target :path))

(defun imoogi-target-scan--target-kind (target)
  "Return TARGET's kind."
  (plist-get target :kind))

(defun imoogi-target-scan--discover-directory (root directory visited)
  "Return Org files below DIRECTORY confined to ROOT.
VISITED is a hash table of canonical directories already traversed.
The return value is a plist (:files FILES :failures FAILURES)."
  (let ((dir (imoogi-target-scan--directory-name directory))
        files failures)
    (condition-case nil
        (let ((true-dir (imoogi-target-scan--true-directory dir)))
          (cond
           ((not (file-directory-p dir))
            (push dir failures))
           ((not (imoogi-target-scan--inside-directory-p true-dir root))
            (push dir failures))
           ((gethash true-dir visited)
            nil)
           (t
            (puthash true-dir t visited)
            (dolist (name (sort (directory-files dir nil "\\`[^.]") #'string<))
              (let ((path (expand-file-name name dir)))
                (cond
                 ((file-directory-p path)
                  (let ((scan (imoogi-target-scan--discover-directory
                               root path visited)))
                    (setq files (nconc files (plist-get scan :files)))
                    (setq failures (nconc failures (plist-get scan :failures)))))
                 ((imoogi-target-scan--org-file-p path)
                  (setq files (nconc files (list path))))))))))
      (file-error (push dir failures))
      (file-missing (push dir failures)))
    (list :files files :failures (nreverse failures))))

(defun imoogi-target-scan--group-files (group)
  "Return discovered files and failures for GROUP."
  (let ((root (plist-get group :root))
        (visited (make-hash-table :test #'equal))
        files failures)
    (when (plist-get group :legacy)
      (let ((scan (imoogi-target-scan--discover-directory root root visited)))
        (setq files (nconc files (plist-get scan :files)))
        (setq failures (nconc failures (plist-get scan :failures)))))
    (dolist (directory (plist-get group :directory-targets))
      (let ((scan (imoogi-target-scan--discover-directory root directory visited)))
        (setq files (nconc files (plist-get scan :files)))
        (setq failures (nconc failures (plist-get scan :failures)))))
    (dolist (file (plist-get group :file-targets))
      (cond
       ((not (file-exists-p file))
        (push file failures))
       ((not (file-regular-p file))
        (push file failures))
       ((imoogi-target-scan--org-file-p file)
        (setq files (nconc files (list file))))))
    (list :files files :failures (append (plist-get group :failures)
                                         (nreverse failures)))))

(defun imoogi-target-scan--scan-file (file root patterns)
  "Scan FILE as part of ROOT using PATTERNS."
  (let* ((relative (file-relative-name file root))
         (excluded (imoogi-scan--excluded-p relative patterns)))
    (imoogi-scan--file file relative excluded)))

(defun imoogi-target-scan--remap-census (census file root)
  "Return CENSUS with each source path remapped for FILE relative to ROOT."
  (let ((relative (file-relative-name file root)))
    (mapcar (lambda (entry)
              (list :note-id (plist-get entry :note-id)
                    :source-path relative))
            census)))

;;;###autoload
(defun imoogi-target-scan (root targets patterns)
  "Scan legacy ROOT and registered TARGETS with PATTERNS.

TARGETS is a list of plists shaped as (:kind directory|file :path PATH).
The return value is a list of group plists.  Each group is shaped as
(:root ABS-DIR :legacy BOOL :scan SCAN-PLIST), where SCAN-PLIST follows
`imoogi-scan-root' with group-local :entries and globally combined
:census, plus :files containing successfully scanned absolute filenames
(including excluded files and files without card headings)."
  (let ((groups nil)
        (covered-directories nil))
    (when root
      (setq groups (imoogi-target-scan--ensure-group groups root t))
      (push (plist-get (car groups) :root) covered-directories))
    (dolist (target (cl-remove-if-not
                     (lambda (target) (eq (imoogi-target-scan--target-kind target) 'directory))
                     targets))
      (let* ((path (imoogi-target-scan--target-path target))
             (dir (imoogi-target-scan--true-directory path)))
        (setq groups (imoogi-target-scan--ensure-group groups dir nil))
        (setq groups (imoogi-target-scan--add-to-group groups dir :directory-targets path))
        (push dir covered-directories)))
    (dolist (target (cl-remove-if-not
                     (lambda (target) (eq (imoogi-target-scan--target-kind target) 'file))
                     targets))
      (let* ((path (expand-file-name (imoogi-target-scan--target-path target)))
             (parent (file-name-directory path))
             (covered (imoogi-target-scan--covered-by-directory-p
                       path covered-directories)))
        (if covered
            (let ((group (imoogi-target-scan--covering-group groups path)))
              (when group
                (setq groups
                      (imoogi-target-scan--add-to-group
                       groups (plist-get group :root) :file-targets path))))
          (setq groups (imoogi-target-scan--ensure-group groups parent nil))
          (setq groups (imoogi-target-scan--add-to-group
                        groups parent :file-targets path)))))
    (let ((seen-files (make-hash-table :test #'equal))
          (global-census nil)
          (group-files nil))
      (dolist (group groups)
        (let* ((root (plist-get group :root))
               (discovery (imoogi-target-scan--group-files group))
               (own-files nil)
               (own-failures (plist-get discovery :failures))
               (own-complete (null own-failures)))
          (dolist (file (plist-get discovery :files))
            (let ((true-file (imoogi-target-scan--true-path file)))
              (cond
               ((not (imoogi-target-scan--inside-directory-p true-file root))
                (push file own-failures)
                (setq own-complete nil))
               ((not (gethash true-file seen-files))
                (puthash true-file t seen-files)
                (condition-case nil
                    (let* ((scan (imoogi-target-scan--scan-file true-file root patterns))
                           (entries (plist-get scan :entries))
                           (census (plist-get scan :census)))
                      (plist-put group :entries
                                 (nconc (plist-get group :entries) entries))
                      (setq global-census
                            (nconc global-census
                                   (list (list :file true-file :census census))))
                      (push true-file own-files))
                  (file-error
                   (push file own-failures)
                   (setq own-complete nil))
                  (file-missing
                   (push file own-failures)
                   (setq own-complete nil)))))))
          (push (list :group group
                      :files (nreverse own-files)
                      :complete own-complete
                      :failures (nreverse own-failures))
                group-files)))
      (setq group-files (nreverse group-files))
      (let ((complete (cl-every (lambda (state) (plist-get state :complete)) group-files)))
        (mapcar
         (lambda (state)
           (let* ((group (plist-get state :group))
                  (root (plist-get group :root))
                  (census (apply #'nconc
                                 (mapcar
                                  (lambda (record)
                                    (imoogi-target-scan--remap-census
                                     (plist-get record :census)
                                     (plist-get record :file)
                                     root))
                                  global-census))))
             (list :root root
                   :legacy (plist-get group :legacy)
                   :scan (list :files (plist-get state :files)
                               :entries (plist-get group :entries)
                               :census census
                               :scan-complete complete
                               :unreadable-files (plist-get state :failures)))))
         group-files)))))

(provide 'imoogi-target-scan)
;;; imoogi-target-scan.el ends here
